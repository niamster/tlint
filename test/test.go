package test

import (
	"fmt"
)

func testPtr() *int {
	return nil
}

func testPtrAndErr() (*int, error) {
	if true {
		return new(int), fmt.Errorf("bang")
	}
	return nil, nil
}

type itf interface {
}

func testItf() itf {
	return nil
}

func testItfAndErr() (itf, error) {
	return nil, nil
}
