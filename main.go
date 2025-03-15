package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println(os.Getpagesize())
	fmt.Println(os.Getpagesize() / 100)
}
