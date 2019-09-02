package main

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"crypto/sha1"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type xvachunk struct {
	b        []byte
	checksum string
	offset   int
}

func main() {
	flag.Parse()
	f, err := os.Open(flag.Args()[0])
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	g, err := gzip.NewReader(f)
	if err != nil {
		log.Fatal(err)
	}
	bufferedReader := bufio.NewReader(g)

	xva := tar.NewReader(bufferedReader)
	lastchunk := xvachunk{}
	if _, err := os.Stat(flag.Args()[1]); err == nil {
		log.Fatal("Output File Already Exists")
	}
	dest, err := os.OpenFile(flag.Args()[1], os.O_CREATE|os.O_WRONLY, os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}
	defer dest.Close()
	x := make(chan xvachunk, 200)
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go writeFile(dest, x, wg, false)
	curdisk := ""
	start := time.Now()
	log.Println("Start Time:", start)
outer:
	for {
		hdr, err := xva.Next()
		switch {

		// if no more files are found return
		case err == io.EOF:
			break outer

		// return any other error
		case err != nil:
			log.Fatal(err)

		// if the header is nil, just skip it (not sure how this happens)
		case hdr == nil:
			continue
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			log.Println("New Disk Found:", hdr.Name)
			if curdisk == "" {
				curdisk = hdr.Name
				continue
			}
			break outer
		case tar.TypeReg:
			if hdr.Name == "ova.xml" {
				b, err := ioutil.ReadAll(xva)
				if err != nil {
					log.Fatal(err)
				}
				ioutil.WriteFile("ova.xml", b, os.ModePerm)
				continue
			}
			disk := strings.Split(hdr.Name, "/")[0]
			if curdisk == "" {
				curdisk = disk
				log.Printf("Writing Disk: %s\n", curdisk)
			}
			if disk != curdisk {
				log.Println("Stopping New Disk Found:", disk)
				break outer
			}
			if !strings.Contains(hdr.Name, ".checksum") {
				//log.Println(strings.Split(hdr.Name, "/")[1])
				chunk, err := strconv.Atoi(strings.Split(hdr.Name, "/")[1])
				if err != nil {
					log.Fatal("Unable to find chunk number", err)
				}
				//log.Println(chunk)
				b, err := ioutil.ReadAll(xva)
				if err != nil {
					log.Fatal(err)
				}
				lastchunk = xvachunk{b: b, offset: chunk}
				continue
			}
			if !strings.Contains(hdr.Name, fmt.Sprintf("%d.checksum", lastchunk.offset)) {
				log.Fatalf("Didnt find expected checksum. Got %s Expected %s", strings.Split(hdr.Name, "/"), fmt.Sprintf("%d.checksum", lastchunk.offset))
			}
			csum, err := ioutil.ReadAll(xva)
			if err != nil {
				log.Fatal(err)
			}
			if len(csum) != 40 {
				log.Fatal("Invalid Checksum Length", len(csum))
			}

			lastchunk.checksum = string(csum)
			x <- lastchunk
		}
	}
	log.Println("Finished Reading XVA")
	close(x)
	log.Println("Waiting for Writing to Finish")
	wg.Wait()
	dest.Close()
	log.Println("Process Complete. Run Time:", time.Since(start))
}

var blankchunk []byte

const blocksize = 1024 * 1024

func writeFile(f io.WriteSeeker, c chan xvachunk, wg *sync.WaitGroup, sparse bool) {
	blankchunk = make([]byte, 1024*1024)
	p := xvachunk{}
	b := 0
	chunks := 0
	empty := 0
	for chk := range c {
		cs := sha1.Sum(chk.b)
		if hex.EncodeToString(cs[:]) != string(chk.checksum) {
			log.Fatalf("Invalid Checksum got %x expected %x", cs, chk.checksum)
		}
		chunks++
		b = b + len(chk.b)
		if p.b == nil { //Its our first chunk
			p = chk
			_, err := f.Write(chk.b)
			if err != nil {
				log.Fatal("Unable to Write Chunk to File")
			}
			continue
		}
		//log.Println("Difference", (chk.offset - p.offset))
		missing := (chk.offset - p.offset) - 1
		if missing >= 1 {
			empty = empty + missing
			if sparse {

				_, err := f.Seek(int64(missing*blocksize), os.SEEK_CUR)
				if err != nil {
					log.Fatal("Unable to Seek", err)
				}
			} else {
				for i := 0; i < missing; i++ {
					//log.Println("Writing Blank Chunk")
					j := 0
					for j < len(blankchunk) {
						w, err := f.Write(blankchunk[j:])
						if err != nil {
							log.Fatal("Unable to Write Blank Chunk to File", err)
						}
						j = j + w
					}
				}
			}
		}
		j := 0
		for j < len(chk.b) {
			w, err := f.Write(chk.b[j:])
			if err != nil {
				log.Fatal("Unable to Write Chunk to File", err)
			}
			j = j + w
		}

		p = chk
	}
	log.Printf("Wrote %d Bytes, Chunks: %d, Empty: %d\n", b, chunks, empty)
	wg.Done()
}
