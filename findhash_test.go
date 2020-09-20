package main

import (
	"hash/fnv"
	"hash/maphash"
	"math/bits"
	"math/rand"
	"testing"
)

func TestHashInputLen(t *testing.T) {
	testcases := []struct {
		cases     []string
		uniqueLen int
	}{
		{[]string{"", "a", "ab"}, 0},
		{[]string{"", "ab", "bb"}, 1},
		{[]string{"abc", "abd", ""}, 3},
		{[]string{"", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "ab"}, 2},
		{[]string{"386", "amd64", "arm"}, 2},
	}

	for _, tc := range testcases {
		ul := minInputLen(tc.cases)
		if ul != tc.uniqueLen {
			t.Errorf("got uniqueLen %d, expected %d for %v", ul, tc.uniqueLen, tc.cases)
		}
	}
}

func TestMPHF(t *testing.T) {
	for _, cases := range testcases {
		m, ok := findMPHF(cases)
		if !ok {
			t.Fatal("could not find MPHF")
		}

		// jump table mask and size
		got := bits.Len32(m.jmpMask)
		expected := bits.Len32(uint32(len(cases)))
		if got != expected {
			t.Errorf("got mask with %d bits, expected %d", got, expected)
		}

		got = bits.OnesCount32(m.jmpMask)
		if got != expected {
			t.Errorf("got mask with %d one-bits, expected %d", got, expected)
		}

		if len(m.jmpTab) <= len(cases) {
			t.Errorf("jump table must have more entries than the cases it codes")
		}
		if len(m.jmpTab) != int(m.jmpMask+1) {
			t.Errorf("got jump table size %d, expected %d", len(m.jmpTab), m.jmpMask+1)
		}

		hasHash := make([]bool, len(m.jmpTab))
		for _, str := range cases {
			hash := m.hashString(str)
			if hash >= uint32(len(m.jmpTab)) {
				t.Errorf("hash(%q)=%d exceeds jump table %d", str, hash, len(m.jmpTab))
				continue
			}
			if hasHash[hash] {
				t.Errorf("hash collision for %q in %+v", str, cases)
			}
			hasHash[hash] = true
		}
	}
}

func BenchmarkFindHash(b *testing.B) {
	var x int
	b.Run("findMPHF", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			findMPHF(testcases[x])
			x = (x + 1) % len(testcases)
		}
	})
}

func BenchmarkJumpTables(b *testing.B) {
	hashes := make([]*mphf, len(testcases))
	for i, cases := range testcases {
		m, ok := findMPHF(cases)
		if !ok {
			b.Error("could not find MPHF")
		}
		hashes[i] = m
	}

	var x, y int
	b.Run("mphf", func(b *testing.B) {
		for k := 0; k < b.N; k++ {
			y++
			if y >= len(testcases[x]) {
				x = (x + 1) % len(testcases)
				y = 0
			}

			_ = hashes[x].jmpTab[hashes[x].hashString(testcases[x][y])]
		}
	})

	maps := make([]map[string]int, len(testcases))
	for i, cases := range testcases {
		maps[i] = make(map[string]int, len(cases))
		for j, key := range cases {
			maps[i][key] = j
		}
	}

	x, y = 0, 0
	b.Run("map", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			y++
			if y >= len(testcases[x]) {
				x = (x + 1) % len(testcases)
				y = 0
			}

			_ = maps[x][testcases[x][y]]
		}
	})
}

func BenchmarkHashes(b *testing.B) {
	hashes := make([]fnv1a, len(testcases))
	for i, cases := range testcases {
		fnv, ok := findHash(cases)
		if !ok {
			b.Error("could not find MPHF")
		}
		hashes[i] = fnv
	}

	var x, y int
	b.Run("minlength fnv1a", func(b *testing.B) {
		for k := 0; k < b.N; k++ {
			y++
			if y >= len(testcases[x]) {
				x = (x + 1) % len(testcases)
				y = 0
			}

			hashes[x].hashString(testcases[x][y])
		}
	})

	x, y = 0, 0
	f := newFnv1a(rand.Uint32(), 1<<30)
	b.Run("full-length fnv1a", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			y++
			if y >= len(testcases[x]) {
				x = (x + 1) % len(testcases)
				y = 0
			}

			f.hashString(testcases[x][y])
		}
	})

	x, y = 0, 0
	var mh maphash.Hash
	b.Run("maphash", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			y++
			if y >= len(testcases[x]) {
				x = (x + 1) % len(testcases)
				y = 0
			}

			mh.WriteString(testcases[x][y])
			mh.Reset()
		}
	})

	x, y = 0, 0
	fnv32 := fnv.New32a()
	b.Run("fnv.New32", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			y++
			if y >= len(testcases[x]) {
				x = (x + 1) % len(testcases)
				y = 0
			}

			fnv32.Write([]byte(testcases[x][y]))
			fnv32.Reset()
		}
	})
}
