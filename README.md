# Prototype code for golang/go #34381

## 1. Perfect hash function

Using FNV (variant 1a for better avalanche properties).

1. Use random seed
2. Hash string length (as one byte);
3. If there are multiple same-length strings: find minimal unique prefix of the
   case strings.
4. If there are collisions, try another seed.

Use 32-bit hash, to fit the hash sum in a word on 32-bit architectures. Could be
extended with 64-bit implementation if needed.

## 2. Minimal perfect hash function (for jump table)

The jump table index is calculated in the following manner, inspired by [0], [1].
The _hash_ function used is the FNV-based perfect hash function mentioned above.

For _N_ pre-defined keys (strings):

1. Define jump table size _m_: the smallest power of 2 greater than _N_
2. Assign the keys to buckets: the number of buckets k is the smallest
   power of 2 greater than _N/3_. Each key is assigned to bucket _hash(key) mod k_
3. Each bucket gets a _shift_ value so that all keys in that bucket get a
   unique jump table index that doesn't collide with any other key:

        sum = hash(key)
        shift = bucketShifts[sum mod k]
        sum' = sum >> shift
        jump table index = (sum' xor sum) mod m

5. If we cannot find a suitable _shift_ for some bucket, try a different seed for
   the hash function.

The number of buckets and jump table size are powers of two so we can use
bitmasks instead of the modulo operator. The number of buckets is based on the
results presented in [0].

References:

[0] F. C. Botelho, D. Belazzougui and M. Dietzfelbinger. Compress, hash and
    displace. In Proceedings of the 17th European Symposium on Algorithms
    (ESA 2009). Springer LNCS, 2009.
    http://cmph.sourceforge.net/papers/esa09.pdf
    
[1] Bob Jenkins: Minimal Perfect Hashing,
    http://www.burtleburtle.net/bob/hash/perfect.html#algo
