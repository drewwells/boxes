package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sort"
	"sync"
)

var (
	jbox, jblock         []byte
	shallowboxes, blocks RubixSlice
	hideOutput           bool
)

func init() {
	resp, err := http.Get("https://s3.amazonaws.com/se-code-challenge/boxes.json")
	if err != nil {
		log.Fatal(err)
	}
	jbox, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	resp, err = http.Get("https://s3.amazonaws.com/se-code-challenge/blocks.json")
	if err != nil {
		log.Fatal(err)
	}
	jblock, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	err = json.Unmarshal(jbox, &shallowboxes)
	if err != nil {
		log.Fatal(err)
	}
	err = json.Unmarshal(jblock, &blocks)
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	// Use go test
	FFD(shallowboxes, blocks)
}

func FFD(shallowboxes, blocks RubixSlice) {

	// Allocate more space for boxes, since we constantly append to it
	boxes := make(RubixSlice, len(shallowboxes), len(shallowboxes)*3)
	copy(boxes, shallowboxes)

	// The input data set has some very large blocks that
	// only fit in some boxes. Optimizing for the use case
	// where only one block fits in one type of box.
	sort.Sort(sort.Reverse(boxes))
	sort.Sort(sort.Reverse(blocks))

	// This may help keep memory thrashing down.
	// avgCap := blocks.Len() / boxes.Len()
	// for i := range boxes {
	// 	boxes[i].Keys = make([]string, 0, avgCap)
	// }

	for i := range blocks {
		for j := range boxes {
			if blocks[i].H > boxes[j].H {
				break
			}
			if fit(blocks[i], boxes[j]) {
				boxes[j].Keys = append(boxes[j].Keys, blocks[i].ID())
				blocks[i].Keys = append(blocks[i].Keys, boxes[j].ID())
				blocks[i].Stored = true

				// Optimistically load blocks in at the bottom of boxes.
				// Moving to the next box if a block doesn't fit.
				// Boxes are continually partitioned after inserting blocks
				// and sorted by height.
				insert := []Rubix{Rubix{
					L: blocks[i].L,
					W: boxes[j].W - blocks[i].W,
					H: blocks[i].H,
				}, Rubix{
					L: boxes[j].L - blocks[i].L,
					W: blocks[i].W,
					H: blocks[i].H,
				}, Rubix{
					L: boxes[j].L,
					W: boxes[j].W,
					H: boxes[j].H - blocks[i].H,
				}}
				// Reorder for maximize space first
				if (boxes[j].W-blocks[i].W)*boxes[j].L <
					(boxes[j].L-blocks[i].L)*boxes[j].W {
					insert[1], insert[0] = insert[0], insert[1]
				}

				// Maintain sorted state, which is slower for this small
				// dataset, but should have improvements as the len(boxes)
				// increases.
				for i := range insert {
					pos := boxes.Search(insert[i])
					boxes = append(boxes[:pos],
						append([]Rubix{insert[i]}, boxes[pos:]...)...)
				}
				break
			}
		}
	}

	var missed RubixSlice
	for i := range blocks {
		if !blocks[i].Stored {
			missed = append(missed, blocks[i])
		}
	}
	jmiss, _ := json.MarshalIndent(
		map[string]RubixSlice{
			fmt.Sprintf("%d blocks did not fit", len(missed)): missed,
		}, "", "  ")
	if !hideOutput {
		fmt.Println(string(jmiss))
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

	bs, _ := json.MarshalIndent(Response{
		Boxmapping:   box.M,
		Blockmapping: block.M,
	}, "", "  ")
	if !hideOutput {
		fmt.Println(string(bs))
	}
}

func fit(a, b Rubix) bool {
	if a.L < b.L &&
		a.H < b.H &&
		a.W < b.W {
		return true
	}
	return false
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
	if r.Blockid != "" {
		return fmt.Sprintf("%s: %f x %f x %f",
			r.ID(), r.W, r.L, r.H)
	} else {
		return fmt.Sprintf("%s: %f x %f x %f Storing: %v",
			r.ID(), r.W, r.L, r.H, r.Keys)
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
	if r[i].H < r[j].H {
		return true
	}
	return false
}

func (rs RubixSlice) Search(r Rubix) int {
	pos := sort.Search(len(rs), func(i int) bool {
		return r.H > rs[i].H
	})
	return pos
}
