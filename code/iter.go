package main

import "iter"

func main() {
	var mySeq iter.Seq[int]
	mySeq = func(yield func(n int) bool) {
		for i := 0; i < 5; i++ {
			if !yield(i) {
				return
			}
		}
	}

	for v := range mySeq {
		println(v)
	}
}
