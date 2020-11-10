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