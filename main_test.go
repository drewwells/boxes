package main

import "testing"

func TestMain(t *testing.T) {

	FFD(shallowboxes, blocks)
}

func BenchmarkMain(b *testing.B) {
	hideOutput = true
	for i := 0; i < b.N; i++ {
		FFD(shallowboxes, blocks)
	}
}
