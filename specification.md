# 修改、改进 RxGo 包

## 目录

- [修改、改进 RxGo 包](#修改改进-rxgo-包)
	- [目录](#目录)
	- [任务概述](#任务概述)
	- [设计说明](#设计说明)
		- [``Debounce``:](#debounce)
		- [``Distinct``:](#distinct)
		- [``ElementAt``:](#elementat)
		- [``First``:](#first)
		- [``IgnoreElements``:](#ignoreelements)
		- [``Last``:](#last)
		- [``Skip``:](#skip)
		- [``SkipLast``:](#skiplast)
		- [``Take``:](#take)
		- [``TakeLast``:](#takelast)
	- [测试结果](#测试结果)
		- [单元测试](#单元测试)
		- [功能测试](#功能测试)


## 任务概述
阅读 ReactiveX 文档，在 [pmlpml/RxGo](https://github.com/pmlpml/rxgo) 的基础上，添加一组新的操作——filtering。<br/>
根据 [Rx 中文文档](https://mcxiaoke.gitbooks.io/rxdocs/content/operators/Sample.html) 的说明，在 filtering.go 中编写添加如下 filtering 操作：
- ``Debounce``：仅在过了一段指定的时间还没发射数据时才发射一个数据
- ``Distinct``：过滤重复的数据项
- ``ElementAt``：只发射指定序列位置的数据项
- ``First``：只发射第一项数据
- ``IgnoreElements``：不发射任何数据，只发射 Observable 的终止通知
- ``Last``：只发射最后一项数据
- ``Skip``：过滤 Observable 发射的前 N 项数据
- ``SkipLast``：过滤 Observable 发射的后 N 项数据
- ``Take``：发射 Observable 发射的前 N 项数据
- ``TakeLast``：发射 Observable 发射的后 N 项数据

&emsp;&emsp;由于 ``Filter`` 在 pmlpml/RxGo/transform.go 中已实现，此处不再重复定义。<br/>
<br/>

## 设计说明

首先自定义错误类型，在过滤操作中涉及到对过滤选项的指定，这可能出现越界错误。因此定义越界错误如下：
```go
var ErrIndexOutOfBoundsException = errors.New("index out of bounds")
```
<br/>

定义过滤操作的流操作符：
```go
type filterOperator struct {
	opFunc func(ctx context.Context, o *Observable, item reflect.Value, out chan interface{}) (end bool)
}
```
<br/>

参考 pmlpml/RxGo 编写对应 filtering 操作的 streamOperator 接口：
```go
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
```
在 pmlpml/RxGo 的代码的基础上增加了对可能的缓存的处理，即在 ``Last``、``SkipLast``、``TakeLast``操作中不能确定的发射数据项被缓存并提交到 ``o.flip``，因此需要在 ``op`` 中将缓存的数据发射出去。<br/>

参考 pmlpml/RxGo 定义过滤器类型的 Observable：
```go
func (parent *Observable) newFilterObservable(name string) (o *Observable) {
	o = newObservable()
	o.Name = name

	parent.next = o
	o.pred = parent
	o.root = parent.root

	o.buf_len = BufferLen
	return o
}
```
<br/>

##### ``Debounce``:
```go
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
		time.Sleep(10 * time.Microsecond)// 实际运行时过滤精度有偏差，给予一定的精度范围
		if elementsCount == count {
			end = o.sendToFlow(ctx, x.Interface(), out)
		}
		return
	}}
	return o
}
```
为指定时间间隔需要传入时间参数—— time.Duration 类型的变量 timespan；设置 Observable 的 goroutine 类型为 ThreadComputing，目的是确保 ``Debounce`` 能够在其它 go 程执行时持续监听输入流；设置计数器 elementsCount，累计发射的数据的数量。
如何判断过了一段指定的时间还没发射数据？使用一个临时变量 count 记录该时间段前已发射数据的数量，与该时间段结束时已发射数据的数量相比（即在该时间段内 elementsCount 可能会更新），如果两个的值相等，说明在该时间段内 elementsCount 没有更新，即没有新的数据发射，此时可以发射一个数据。<br/>

##### ``Distinct``:
```go
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
```
使用一个 map 记录已出现过的数据，如果当前输入流中的数据是首次出现，则为其创建一个记录并将其发射出去。<br/>

##### ``ElementAt``:
```go
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
```
设置计数器 elementsCount，累计进入输入流的数据的数量，当某个数据出现使其恰好等于指定的项序号时，发射该数据。**注意**，如果指定的项序号为负数，则抛出一个``ErrIndexOutOfBoundsException``异常。<br/>

##### ``First``:
```go
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
```
使用 firstFlag 标记是否为第一项数据。初始值为 true，直到第一个数据出现将其置为 false，故只有第一个数据能够被发射出去。<br/>

##### ``IgnoreElements``:
```go
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
```
由于不发射任何数据，所以即使输入流有数据出现时也都不作处理。<br/>

##### ``Last``:
```go
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
```
在 opFunc 中我们并不知道输入流在之后是否还有数据出现，即无法在过滤操作时直接确定哪一项数据是最后一项，因此需要缓存当前从输入流中输入的最后一项数据，在 opFunc 返回时再进行处理。<br/>

##### ``Skip``:
```go
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
```
设置计数器 elementsCount，累计进入输入流的数据的数量，当已出现的数据数量小于指定的项数 N 时，不发射当前数据，直到满足 ``elementsCount > index`` 后进入的数据才能被发射。<br/>

##### ``SkipLast``:
```go
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
```
与 ``Last`` 类似的，需要对数据进行缓存才能得到最终能够被发射出去的数据。具体实现是当输入流有新的数据出现时，使用 ``queue`` 使其总是缓存最后出现的至多 N 项数据，当已出现的数据数量已超过 ``queue`` 的长度时，将 ``queue`` 的头部加入到 ``okToSend`` 中，最后 ``okToSend`` 中缓存的就是最终能够被发射出去的数据。<br/>

##### ``Take``:
```go
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
```
设置计数器 elementsCount，累计进入输入流的数据的数量，仅当已出现的数据数量不大于指定的项数 N 时，才能发射当前数据。<br/>

##### ``TakeLast``:
```go
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
```
与 ``SkipLast`` 类似的，使用 ``okToSend`` 使其总是缓存最后出现的至多 N 项数据，最后 ``okToSend`` 中缓存的就是最终能够被发射出去的数据。<br/>
<br/>

## 测试结果

### 单元测试

分别测试已定义的 filtering 操作函数如下：
```go
package rxgo

import(
	"testing"
	"reflect"
	"time"
)

func TestDebounce(t *testing.T) {
	res := []int{}
	Just(1, 2, 3, 4, 5, 6, 7, 8).Map(func(x int) int{
		time.Sleep(1 * time.Millisecond)
		return x
	}).Debounce(2 * time.Millisecond).Subscribe(func(x int) {
		res = append(res, x)
	})

	if !reflect.DeepEqual(res, []int{8}) {
		t.Errorf("debounce filtering error")
	}
}

func TestDistinct(t *testing.T) {
	res := []int{}
	Just(5, 8, 8, 3, 24, 3, 32, 8, 1, 5).Distinct().Subscribe(func(x int) {
		res = append(res, x)
	})

	if !reflect.DeepEqual(res, []int{5, 8, 3, 24, 32, 1}) {
		t.Errorf("distinct filtering error")
	}
}

func TestElementAt(t *testing.T) {
	res := []int{}
	Just(5, 8, 8, 3, 24, 3, 32, 8, 1, 5).ElementAt(6).Subscribe(func(x int) {
		res = append(res, x)
	})

	if !reflect.DeepEqual(res, []int{32}) {
		t.Errorf("elementAt filtering error")
	}
}

func TestFirst(t *testing.T) {
	res := []int{}
	Just(5, 8, 8, 3, 24, 3, 32, 8, 1, 5).First().Subscribe(func(x int) {
		res = append(res, x)
	})

	if !reflect.DeepEqual(res, []int{5}) {
		t.Errorf("first filtering error")
	}
}

func TestIgnoreElements(t *testing.T) {
	res := []int{}
	Just(5, 8, 8, 3, 24, 3, 32, 8, 1, 5).IgnoreElements().Subscribe(func(x int) {
		res = append(res, x)
	})

	if !reflect.DeepEqual(res, []int{}) {
		t.Errorf("ignoreElements filtering error")
	}
}

func TestLast(t *testing.T) {
	res := []int{}
	Just(5, 8, 8, 3, 24, 3, 32, 8, 1, 20).Last().Subscribe(func(x int) {
		res = append(res, x)
	})

	if !reflect.DeepEqual(res, []int{20}) {
		t.Errorf("last filtering error")
	}
}

func TestSkip(t *testing.T) {
	res := []int{}
	Just(5, 8, 8, 3, 24, 3, 32, 8, 1, 5).Skip(4).Subscribe(func(x int) {
		res = append(res, x)
	})

	if !reflect.DeepEqual(res, []int{24, 3, 32, 8, 1, 5}) {
		t.Errorf("skip filtering error")
	}
}

func TestSkipLast(t *testing.T) {
	res := []int{}
	Just(5, 8, 8, 3, 24, 3, 32, 8, 1, 5).SkipLast(4).Subscribe(func(x int) {
		res = append(res, x)
	})

	if !reflect.DeepEqual(res, []int{5, 8, 8, 3, 24, 3}) {
		t.Errorf("skipLast filtering error")
	}
}

func TestTake(t *testing.T) {
	res := []int{}
	Just(5, 8, 8, 3, 24, 3, 32, 8, 1, 5).Take(4).Subscribe(func(x int) {
		res = append(res, x)
	})

	if !reflect.DeepEqual(res, []int{5, 8, 8, 3}) {
		t.Errorf("take filtering error")
	}
}

func TestTakeLast(t *testing.T) {
	res := []int{}
	Just(5, 8, 8, 3, 24, 3, 32, 8, 1, 5).TakeLast(4).Subscribe(func(x int) {
		res = append(res, x)
	})

	if !reflect.DeepEqual(res, []int{32, 8, 1, 5}) {
		t.Errorf("takeLast filtering error")
	}
}
```
测试结果如下：<br/>
![](https://cdn.jsdelivr.net/gh/sherryjw/StaticResource@master/image/sc-hw6-001.png)<br/>
<br/>

### 功能测试

编写 main.go 文件如下：
```go
package main

import (
	"fmt"
	"time"
	rxgo "github.com/sherryjw/rxgo"
)

func main() {

	rxgo.Just(1, 2, 3, 4, 5, 6, 7, 8).Map(func(x int) int{
		time.Sleep(1 * time.Millisecond)
		return x
	}).Debounce(2 * time.Millisecond).Subscribe(func(x int) {
		fmt.Print(x)
		fmt.Print(" ")
	})
	fmt.Println()

	rxgo.Just(5, 8, 8, 3, 24, 3, 32, 8, 1, 5).Distinct().Subscribe(func(x int){
		fmt.Print(x)
		fmt.Print(" ")
	})
	fmt.Println()
	
	rxgo.Just(5, 8, 8, 3, 24, 3, 32, 8, 1, 5).ElementAt(6).Subscribe(func(x int){
		fmt.Print(x)
		fmt.Print(" ")
	})
	fmt.Println()
	
	rxgo.Just(5, 8, 8, 3, 24, 3, 32, 8, 1, 5).First().Subscribe(func(x int){
		fmt.Print(x)
		fmt.Print(" ")
	})
	fmt.Println()
	
	rxgo.Just(5, 8, 8, 3, 24, 3, 32, 8, 1, 5).IgnoreElements().Subscribe(func(x int){
		fmt.Print(x)
		fmt.Print(" ")
	})
	fmt.Println()
	
	rxgo.Just(5, 8, 8, 3, 24, 3, 32, 8, 1, 20).Last().Subscribe(func(x int){
		fmt.Print(x)
		fmt.Print(" ")
	})
	fmt.Println()
	
	rxgo.Just(5, 8, 8, 3, 24, 3, 32, 8, 1, 5).Skip(4).Subscribe(func(x int){
		fmt.Print(x)
		fmt.Print(" ")
	})
	fmt.Println()
	
	rxgo.Just(5, 8, 8, 3, 24, 3, 32, 8, 1, 5).SkipLast(4).Subscribe(func(x int){
		fmt.Print(x)
		fmt.Print(" ")
	})
	fmt.Println()
	
	rxgo.Just(5, 8, 8, 3, 24, 3, 32, 8, 1, 5).Take(4).Subscribe(func(x int){
		fmt.Print(x)
		fmt.Print(" ")
	})
	fmt.Println()
	
	rxgo.Just(5, 8, 8, 3, 24, 3, 32, 8, 1, 5).TakeLast(4).Subscribe(func(x int){
		fmt.Print(x)
		fmt.Print(" ")
	})
	fmt.Println()
}
```
运行：<br/>
![](https://cdn.jsdelivr.net/gh/sherryjw/StaticResource@master/image/sc-hw6-002.png)<br/>
这与我们的预期结果符合！<br/>