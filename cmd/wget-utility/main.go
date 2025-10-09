package main

import (
	"flag"
	"fmt"
	"go-utility/utils/crawler"
)

func main() {
	url := flag.String("url", "", "url link for download")
	depth := flag.Int("d", 1, "depth of recursion")
	output := flag.String("o", "mirror", "Dir for load")

	flag.Parse()

	if *url == "" {
		return
	}

	c := crawler.New(*url, *output, *depth)
	if err := c.Start(); err != nil {
		fmt.Println("Error:", err)
	}
}
