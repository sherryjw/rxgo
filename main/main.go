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
