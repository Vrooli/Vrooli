package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"image-tools/internal/aigateway"
)

func main() {
	path := os.Args[1]
	img, err := os.ReadFile(path)
	if err != nil { panic(err) }
	judge := aigateway.NewAgeJudge(aigateway.NewVisionClient())
	start := time.Now()
	got, err := judge.Judge(context.Background(), img)
	fmt.Printf("elapsed=%s judgement=%+v err=%v\n", time.Since(start).Round(time.Second), got, err)
}
