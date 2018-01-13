package main

import (
	"sync"
)

func FatalErr(err error) {
	if err != nil {
		Log.Fatalln(err.Error())
	}
}

func WaitForTERM() {
	wg := sync.WaitGroup{}
	wg.Add(1)
	wg.Wait()
}
