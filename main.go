package main

import (
	"course/raft"
	"fmt"
)

func main() {
	off := make(chan struct{})
	r, _ := raft.NewNode(101, off)
	fmt.Printf("R: %v\n", r)
}
