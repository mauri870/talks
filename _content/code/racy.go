package main

import (
	"fmt"
	"sync"
)

// START OMIT
func main() {
	var wg sync.WaitGroup
	s := []int{}

	for i := range 10 {
		wg.Add(1)

		go func(ii int) {
			defer wg.Done()

			// do work!
			ii *= 2

			s = append(s, ii)
		}(i)
	}

	wg.Wait()
	fmt.Printf("%v\n", s)
	fmt.Printf("%d", len(s))
}

// END OMIT
