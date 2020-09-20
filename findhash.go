package main

import (
	"fmt"
	"math/rand"
	"sort"
	"time"
)

const maxAttempts = 100 // maximum amount of seeds to try

// findHash tries seeds until it finds a perfect hash function.
// Returns true if found, false if no success after maxAttempts iterations.
func findHash(cases []string) (fnv1a, bool) {
	// Prepare input data
	cases = deduplicate(cases)
	strlen := minInputLen(cases)

	for i := 0; i < maxAttempts; i++ {
		fnv := newFnv1a(rand.Uint32(), strlen)
		if !hasCollisions(cases, fnv) {
			return fnv, true
		}
	}
	return fnv1a{}, false
}

// deduplicate sorts and discards duplicates from data
func deduplicate(data []string) []string {
	sort.Strings(data)
	j := 0
	for i := 1; i < len(data); i++ {
		if data[j] == data[i] {
			// skip duplicate
			continue
		}

		j++
		data[j] = data[i]
	}
	return data[:j+1]
}

// minInputLen finds the minimal length that uniquely identifies a case string
// Return 0 if [string length modulo 256] is unique for each string. Otherwise return the
// minimum number of bytes required to uniquely identify each case.
func minInputLen(cases []string) int {
	// Check if string lengths mod 256 are unique to each case
	lengths := make(map[byte]struct{})
	for _, str := range cases {
		lengths[byte(len(str))] = struct{}{}
	}
	if len(lengths) == len(cases) {
		// All cases have unique lengths
		return 0
	}

	sort.Strings(cases)
	uniqueLen := 0
	for i := 1; i < len(cases); i++ {
		a, b := cases[i-1], cases[i]
		n := 0
		for n < len(a) && n < len(b) && a[n] == b[n] {
			n++
		}
		n++ // convert index to string length

		if n > uniqueLen {
			uniqueLen = n
		}
	}
	return uniqueLen
}

// hasCollisions returns true if fnv hashes collide for any two cases
func hasCollisions(cases []string, fnv fnv1a) bool {
	hashes := make(map[uint32]struct{})

	for _, str := range cases {
		sum := fnv.hashString(str)
		if _, exists := hashes[sum]; exists {
			return true
		}
		hashes[sum] = struct{}{}
	}
	return false
}

const (
	// FNV-1a 32-bit parameters
	offset32 = 2166136261
	prime32  = 16777619
)

// fnv1a is used to calculate the FNV-1a 32-bit hash
type fnv1a struct {
	offset uint32 // seeded initial sum
	strlen int    // maximum bytes to hash
}

// newFnv1a returns a seeded fnv1a
func newFnv1a(seed uint32, strlen int) fnv1a {
	f := fnv1a{offset32, strlen}
	// Hash the seed into f.offset
	for _, w := range []int{0, 8, 16, 24} {
		f.offset = f.hashByte(f.offset, byte(seed>>w))
	}

	return f
}

// hashByte returns the sum hashed with the data.
func (_ fnv1a) hashByte(sum uint32, data byte) uint32 {
	// FNV-1a:
	sum ^= uint32(data)
	sum *= prime32
	return sum
}

// hashString hashes first the length of the string, truncated to one byte, and
// then up to strlen bytes, or to the end of the string. Whichever comes first.
func (f fnv1a) hashString(input string) uint32 {
	// Truncate string length to one byte and hash it
	sum := f.hashByte(f.offset, byte(len(input)))

	// Hash input[:f.strlen]
	for i := 0; i < len(input) && i < f.strlen; i++ {
		sum = f.hashByte(sum, input[i])
	}
	return sum
}

// findMPHF tries seeds until it finds a near minimal perfect hash function.
// Returns true if found, false if no success after maxAttempts iterations.
func findMPHF(cases []string) (*mphf, bool) {
	// Prepare input data
	cases = deduplicate(cases)

	for i := 0; i < maxAttempts; i++ {
		fnv, ok := findHash(cases)
		//fnv := newFnv1a(rand.Uint32(), strlen)
		//if !hasCollisions(cases, fnv) {
		if ok {
			m, ok := newMPHF(cases, fnv)
			if ok {
				return m, true
			}
		}
	}
	return nil, false
}

// mphf is a (near) minimal perfect hash function used for a jump table.
//
// The jump table index is calculated in the following manner, inspired by [0], [1].
//
//     For N pre-defined keys (strings):
//     1. Define jump table size m: the smallest power of 2 greater than N
//     2. Assign the keys to buckets: the number of buckets k is the smallest
//        power of 2 greater than N/3 bucket(key) = hash(key) mod k
//     3. Each bucket gets a shift value so that all keys in that bucket get a
//        unique jump table index that doesn't collide with any other key:
//         sum = hash(key)
//         shift = bucketShifts[sum mod k]
//         sum' = sum >> shift
//         jump table index = (sum' xor sum) mod m
//
// References:
// [0] F. C. Botelho, D. Belazzougui and M. Dietzfelbinger. Compress, hash and
//     displace. In Proceedings of the 17th European Symposium on Algorithms
//     (ESA 2009). Springer LNCS, 2009.
//     http://cmph.sourceforge.net/papers/esa09.pdf
// [1] Bob Jenkins: Minimal Perfect Hashing,
//     http://www.burtleburtle.net/bob/hash/perfect.html#algo

type mphf struct {
	fnv      fnv1a
	bktShift []byte
	bktMask  uint32
	jmpTab   []jmpEntry
	jmpMask  uint32
}

// jmpIx calculates the jump table index for a fnv hash sum
func (m mphf) jmpIx(sum uint32, shift byte) uint32 {
	return ((sum >> shift) ^ sum) & m.jmpMask
}

// hashString calculates the near minimal perfect hash sum for data
func (m mphf) hashString(data string) uint32 {
	sum := m.fnv.hashString(data)
	return m.jmpIx(sum, m.bktShift[sum&m.bktMask])
}

// newMPHF returns a near minimal perfect hash function for the data set.
// Returns false if it was not possible to construct the mphf with this fnv
// hash function.
func newMPHF(cases []string, fnv fnv1a) (*mphf, bool) {
	var m mphf
	m.fnv = fnv

	// Desired jump table size is the smallest power of 2 greater than N
	jmpSize := 1
	for jmpSize <= len(cases) {
		jmpSize <<= 1
	}
	m.jmpTab = make([]jmpEntry, jmpSize)
	m.jmpMask = uint32(jmpSize - 1)

	// Desired number of buckets is the smallest power of 2 greater than N/3
	bucketCnt := 1
	for bucketCnt <= len(cases)/3 {
		bucketCnt <<= 1
	}
	m.bktMask = uint32(bucketCnt - 1)
	m.bktShift = make([]byte, bucketCnt)

	ok := m.initBuckets(cases)
	if !ok {
		return nil, false
	}

	for _, str := range cases {
		m.jmpTab[m.hashString(str)] = jmpEntry{str, true}
	}
	return &m, true
}

// initBuckets initializes the bktShift for each bucket.
// Returns true if we found good shift values for all buckets.
func (m *mphf) initBuckets(cases []string) bool {
	// Populate the hash sums into buckets
	buckets := make([][]uint32, len(m.bktShift))
	for _, str := range cases {
		sum := m.fnv.hashString(str)
		buckets[sum&m.bktMask] = append(buckets[sum&m.bktMask], sum)
	}

	// Sort by bucket size, largest first
	sort.Slice(buckets, func(i, j int) bool {
		return len(buckets[i]) > len(buckets[j])
	})

	// Find a shift value for each bucket
	hasJump := make([]bool, len(m.jmpTab))
	for _, sums := range buckets {
		if len(sums) == 0 {
			break
		}

		// Find a shift value for this bucket so that all sums in this bucket
		// avoid collisions in the jump table.
		foundShift := false
		for shift := byte(0); shift < 32; shift++ {
			shiftOk := true
			newJump := make([]bool, len(m.jmpTab))

			// Try placing sums in the jump table
			for _, sum := range sums {
				ix := m.jmpIx(sum, shift)
				if hasJump[ix] || newJump[ix] {
					// Collision in the jump table, cannot use this shift value
					shiftOk = false
					break
				}
				newJump[ix] = true
			}

			if shiftOk {
				// Found a valid shift value for this bucket
				foundShift = true
				m.bktShift[sums[0]&m.bktMask] = shift
				for ix, addJump := range newJump {
					if addJump {
						hasJump[ix] = true
					}
				}
				break
			}
		}
		if !foundShift {
			return false
		}
	}
	return true
}

type jmpEntry struct {
	key   string
	valid bool
}

func main() {
	var mphfs int
	var successCnt int
	var total int

	start := time.Now()
	for _, cases := range testcases {
		_, ok := findMPHF(cases)
		if ok {
			successCnt++
			mphfs++
		}
		total++
	}
	end := time.Now()

	fmt.Printf("Success rate: %.1f%%\n", 100*float64(successCnt)/float64(total))
	fmt.Printf("MPHF rate: %.1f%%\n", 100*float64(mphfs)/float64(total))
	fmt.Println("Total time:", end.Sub(start))
}
