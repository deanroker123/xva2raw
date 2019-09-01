package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
)

var offset = flag.Int64("offset", 0, "Starting offset in MB")

func main() {
	flag.Parse()
	log.Println("Starting offset", *offset)
	file1 := flag.Arg(0)
	file2 := flag.Arg(1)
	if file1 == "" || file2 == "" {
		log.Fatal("Must supply 2 files to compare")
	}
	f1, err := os.Open(file1)
	if err != nil {
		log.Fatal(err)
	}
	f2, err := os.Open(file2)
	if err != nil {
		log.Fatal(err)
	}
	block := *offset
	f1read, f2read := 0, 0
	f1.Seek(1024*1024**offset, 0)
	f2.Seek(1024*1024**offset, 0)
	chan1 := make(chan []byte, 2)
	chan2 := make(chan []byte, 2)
	go readfile(1, f1, chan1)
	go readfile(2, f2, chan2)
	//defer log.Printf("Blocks: %d, f1 bytes: %d, f2 bytes:%d\n", block, f1read, f2read)
	for {

		b1, ok := <-chan1
		if !ok {
			log.Println("B1 not ok")
			log.Printf("Blocks: %d, f1 bytes: %d, f2 bytes:%d\n", block, f1read, f2read)
			break
		}
		f1read = f1read + len(b1)
		b2, ok := <-chan2
		if !ok {
			log.Println("B2 not ok")
			log.Printf("Blocks: %d, f1 bytes: %d, f2 bytes:%d\n", block, f1read, f2read)
			break
		}
		f2read = f2read + len(b2)
		if !testEq(b1, b2) {
			log.Printf("Block %d does not match", block)
			ioutil.WriteFile(fmt.Sprintf("f1_%d", block), b1, os.ModePerm)
			ioutil.WriteFile(fmt.Sprintf("f2_%d", block), b2, os.ModePerm)
		}

		
		block++
		if block%1000 == 0 {
			log.Printf("Blocks: %d, f1 bytes: %d, f2 bytes:%d\n", block, f1read, f2read)
		}
	}
}

func readfile(n int, f *os.File, c chan []byte) {

	for {
		b := make([]byte, 1024*1024)
		i, err := f.Read(b)
		if err == io.EOF {

			break
		}
		if err != nil {
			log.Fatal(err)
		}
		if i < len(b) {
			log.Fatalf("%d Didnt read enough", n)
		}
		c <- b[:i]
	}
	log.Printf("File %d EOF", n)
	close(c)
}

func testEq(a, b []byte) bool {

	// If one is nil, the other must also be nil.
	if (a == nil) != (b == nil) {
		return false
	}
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
