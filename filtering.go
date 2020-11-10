package rxgo

import (
	"fmt"
	"context"
	"reflect"
	"sync"
	"errors"
	"time"
)

// if user function throw IndexOutOfBoundsException, the Observable will stop and close it
var ErrIndexOutOfBoundsException = errors.New("index out of bounds")

type filterOperator struct {
	opFunc func(ctx context.Context, o *Observable, item reflect.Value, out chan interface{}) (end bool)
}

func (ftop filterOperator) op(ctx context.Context, o *Observable) {
	in := o.pred.outflow
	out := o.outflow

	var wg sync.WaitGroup

	go func() {
		end := false
		for x := range in {
			if end {
				continue
			}

			xv := reflect.ValueOf(x)

			if e, ok := x.(error); ok && !o.flip_accept_error {
				o.sendToFlow(ctx, e, out)
				continue
			}

			switch threading := o.threading; threading {
			case ThreadingDefault:
				if ftop.opFunc(ctx, o, xv, out) {
					end = true
				}
			case ThreadingIO:
				fallthrough
			case ThreadingComputing:
				wg.Add(1)
				go func() {
					defer wg.Done()
					if ftop.opFunc(ctx, o, xv, out) {
						end = true
					}
				}()
			default:
			}
		}

		if o.flip != nil {
			buffer, _ := reflect.ValueOf(o.flip).Interface().([]reflect.Value)
			for _, v := range buffer {
				o.sendToFlow(ctx, v.Interface(), out)
			}
		}

		wg.Wait()
		o.closeFlow(out)
	}()

}

func (parent *Observable) newFilterObservable(name string) (o *Observable) {
	o = newObservable()
	o.Name = name

	parent.next = o
	o.pred = parent
	o.root = parent.root

	o.buf_len = BufferLen
	return o
}

func (parent *Observable) Debounce(timespan time.Duration) (o *Observable) {
	o = parent.newFilterObservable("debounce")
	o.flip = nil
	o.flip_sup_ctx = true
	o.flip_accept_error = true
	o.threading = ThreadingComputing

	elementsCount := 0
	o.operator = filterOperator{func(ctx context.Context, o *Observable, x reflect.Value, out chan interface{}) (end bool){
		count := elementsCount
		count++
		elementsCount++
		time.Sleep(timespan)
		time.Sleep(10 * time.Microsecond)// cu lue di guo lv
		if elementsCount == count {
			end = o.sendToFlow(ctx, x.Interface(), out)
		}
		return
	}}
	return o
}

func (parent *Observable) Distinct() (o *Observable) {
	o = parent.newFilterObservable("distinct")
	o.flip = nil
	o.flip_sup_ctx = true
	o.flip_accept_error = true

	mapForDistinct := make(map[string]bool)
	o.operator = filterOperator{func(ctx context.Context, o *Observable, x reflect.Value, out chan interface{}) (end bool){
		element := fmt.Sprintf("%v", x)
		if _, recorded := mapForDistinct[element]; !recorded {
			mapForDistinct[element] = true
			end = o.sendToFlow(ctx, x.Interface(), out)
		}
		return
	}}
	return o
}

func (parent *Observable) ElementAt(N int) (o *Observable) {
	o = parent.newFilterObservable("elementAt")
	o.flip = nil
	o.flip_sup_ctx = true
	o.flip_accept_error = true

	elementsCount := 0
	o.operator = filterOperator{func(ctx context.Context, o *Observable, x reflect.Value, out chan interface{}) (end bool){
		elementsCount++
		if N < 0 {
			end = o.sendToFlow(ctx, ErrIndexOutOfBoundsException, out)
		}
		if elementsCount == N+1 {
			end = o.sendToFlow(ctx, x.Interface(), out)
		}
		return
	}}
	return o
}

func (parent *Observable) First() (o *Observable) {
	o = parent.newFilterObservable("first")
	o.flip = nil
	o.flip_sup_ctx = true
	o.flip_accept_error = true

	firstFlag := true
	o.operator = filterOperator{func(ctx context.Context, o *Observable, x reflect.Value, out chan interface{}) (end bool){
		if firstFlag {
			end = o.sendToFlow(ctx, x.Interface(), out)
			firstFlag = false
		}
		return
	}}
	return o
}

func (parent *Observable) IgnoreElements() (o *Observable) {
	o = parent.newFilterObservable("ignoreElements")
	o.flip = nil
	o.flip_sup_ctx = true
	o.flip_accept_error = true

	o.operator = filterOperator{func(ctx context.Context, o *Observable, x reflect.Value, out chan interface{}) (end bool){
		return
	}}
	return o
}

func (parent *Observable) Last() (o *Observable) {
	o = parent.newFilterObservable("last")
	o.flip = nil
	o.flip_sup_ctx = true
	o.flip_accept_error = true

	last := make([]reflect.Value, 1)
	o.operator = filterOperator{func(ctx context.Context, o *Observable, x reflect.Value, out chan interface{}) (end bool){
		last[0] = x
		o.flip = last
		return
	}}
	return o
}

func (parent *Observable) Skip(N int) (o *Observable) {
	o = parent.newFilterObservable("skip")
	o.flip = nil
	o.flip_sup_ctx = true
	o.flip_accept_error = true

	elementsCount := 0	
	o.operator = filterOperator{func(ctx context.Context, o *Observable, x reflect.Value, out chan interface{}) (end bool){
		elementsCount++
		if N < 0 {
			end = o.sendToFlow(ctx, ErrIndexOutOfBoundsException, out)
		}
		if elementsCount > N {
			end = o.sendToFlow(ctx, x.Interface(), out)
		}
		return
	}}
	return o
}

func (parent *Observable) SkipLast(N int) (o *Observable) {
	o = parent.newFilterObservable("skipLast")
	o.flip = nil
	o.flip_sup_ctx = true
	o.flip_accept_error = true

	okToSend := make([]reflect.Value, 0)
	queue := make([]reflect.Value, 0)

	o.operator = filterOperator{func(ctx context.Context, o *Observable, x reflect.Value, out chan interface{}) (end bool){
		queue = append(queue, x)
		l := len(queue)
		if l > N {
			okToSend = append(okToSend, queue[0])
			queue = queue[1: l]
		}
		o.flip = okToSend
		return
	}}
	return o
}

func (parent *Observable) Take(N int) (o *Observable) {
	o = parent.newFilterObservable("take")
	o.flip = nil
	o.flip_sup_ctx = true
	o.flip_accept_error = true

	elementsCount := 0
	o.operator = filterOperator{func(ctx context.Context, o *Observable, x reflect.Value, out chan interface{}) (end bool){
		elementsCount++
		if N < 0 {
			end = o.sendToFlow(ctx, ErrIndexOutOfBoundsException, out)
		}
		if elementsCount <= N {
			end = o.sendToFlow(ctx, x.Interface(), out)
		}
		return
	}}
	return o
}

func (parent *Observable) TakeLast(N int) (o *Observable) {
	o = parent.newFilterObservable("takeLast")
	o.flip = nil
	o.flip_sup_ctx = true
	o.flip_accept_error = true

	okToSend := make([]reflect.Value, N)
	o.operator = filterOperator{func(ctx context.Context, o *Observable, x reflect.Value, out chan interface{}) (end bool){
		okToSend = append(okToSend, x)
		l := len(okToSend)
		if l > N {
			okToSend = okToSend[1: l]
		}
		o.flip = okToSend
		return
	}}
	return o
}