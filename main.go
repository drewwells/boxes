package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"sort"
	"sync"
)

func main() {

	jbox, _ := ioutil.ReadFile("boxes.json")
	jblock, _ := ioutil.ReadFile("blocks.json")
	var (
		boxes, blocks RubixSlice
	)
	err := json.Unmarshal(jbox, &boxes)
	if err != nil {
		log.Fatal(err)
	}
	err = json.Unmarshal(jblock, &blocks)
	if err != nil {
		log.Fatal(err)
	}

	sort.Sort(boxes)
	sort.Sort(sort.Reverse(blocks))

	// Initialize the theoretical average number of boxes per block
	// This may help keep memory thrashing down.
	// avgCap := blocks.Len() / boxes.Len()
	// for i := range boxes {
	// 	boxes[i].Keys = make([]string, 0, avgCap)
	// }

	for i := range blocks {
		for j := range boxes {
			if blocks[i].L < boxes[j].L && blocks[i].H < boxes[j].H && blocks[i].W < boxes[j].W {
				// Create three new boxes for the remaining space (if any)
				boxes[j].Keys = append(boxes[j].Keys, blocks[i].ID())
				blocks[i].Keys = append(blocks[i].Keys, boxes[j].ID())
				blocks[i].Stored = true

				// L
				boxes = append(boxes, Rubix{
					L:     boxes[j].L - blocks[i].L,
					W:     boxes[j].W,
					H:     boxes[j].H,
					Boxid: boxes[j].Boxid,
				})

				// H
				boxes = append(boxes, Rubix{
					L:     blocks[i].L,
					W:     boxes[j].W,
					H:     boxes[j].H - blocks[i].H,
					Boxid: boxes[j].Boxid,
				})

				// H
				boxes = append(boxes, Rubix{
					L:     blocks[i].L,
					W:     boxes[j].W - blocks[i].W,
					H:     blocks[i].H,
					Boxid: boxes[j].Boxid,
				})

				sort.Sort(boxes)
				break
			}
		}
	}

	// Check if any were missed
	for i := range blocks {
		if !blocks[i].Stored {
			fmt.Println("Could not fit:", blocks[i])
		}
	}

	box := SafeBox{}
	box.M = make(map[string][]string, len(boxes))
	for i := range boxes {
		id := boxes[i].ID()
		box.M[id] = append(box.M[id], boxes[i].Keys...)
	}

	block := SafeBlock{}
	block.M = make(map[string]string, len(blocks))
	for i := range blocks {
		id := blocks[i].ID()
		if len(blocks[i].Keys) > 0 {
			block.M[id] = blocks[i].Keys[0]
		}
	}

	r := Response{
		Boxmapping:   box.M,
		Blockmapping: block.M,
	}
	bs, err := json.Marshal(r)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(bs))
}

type Response struct {
	Boxmapping   map[string][]string
	Blockmapping map[string]string
}
type SafeBox struct {
	sync.Mutex
	M map[string][]string
}

type SafeBlock struct {
	sync.Mutex
	M map[string]string
}

type Rubix struct {
	L       float64 `json:"length"`
	W       float64 `json:"width"`
	H       float64 `json:"height"`
	Blockid string  `json:"blockid"`
	Boxid   string  `json:"boxid"`
	Keys    []string
	Stored  bool
}

func (r Rubix) String() string {
	if r.Boxid != "" {
		return fmt.Sprintf("%s: %fx%fx%f", r.ID(), r.H, r.W, r.L)
	} else {
		return fmt.Sprintf("%s: %f Storing: %v", r.ID(), r.Size(), r.Keys)
	}
}

func (r Rubix) ID() string {
	if r.Blockid == "" {
		return r.Boxid
	}
	return r.Blockid
}

func (r Rubix) Size() float64 {
	return r.L * r.W * r.H
}

type RubixSlice []Rubix

func (r RubixSlice) Len() int {
	return len(r)
}

func (r RubixSlice) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}

func (r RubixSlice) Less(i, j int) bool {
	if r[i].Size() < r[j].Size() {
		return true
	}
	return false
}
