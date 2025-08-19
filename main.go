package main

import (
	"fmt"
	"runtime"
)

func main() {
	fmt.Println("Hello, World from PowerShell!")
	fmt.Printf("Go version: %s\n", goVersion())
}

func goVersion() string {
	return runtime.Version()
}
