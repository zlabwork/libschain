package bitcoin

import (
	"bytes"
	"crypto/rand"
	"github.com/FactomProject/btcutilecc"
	"github.com/mr-tron/base58"
	"math/big"
)

var (
	curve                     = btcutil.Secp256k1()
	PublicKeyCompressedLength = 33
)

// https://learnmeabitcoin.com/technical/private-key
func GeneratePrivateKey() []byte {
	// TODO :: 私钥值约束，最大不能大于 fffffffffffffffffffffffffffffffebaaedce6af48a03bbfd25e8cd0364140
	b := make([]byte, 32)
	rand.Read(b)
	return b
}

// https://learnmeabitcoin.com/technical/public-key
func GeneratePublicKey(priKey []byte) []byte {
	curve.ScalarBaseMult(priKey)
	return compressPublicKey(curve.ScalarBaseMult(priKey))
}

func GenerateUncompressedPublicKey(priKey []byte) []byte {
	curve.ScalarBaseMult(priKey)
	return uncompressedPublicKey(curve.ScalarBaseMult(priKey))
}

// https://learnmeabitcoin.com/technical/wif
func WIF(priKey []byte) string {
	version := []byte{0x80}
	compression := byte(0x01)
	key := append(version, priKey...)
	key = append(key, compression)
	sum, err := checksum(key)
	if err != nil {
		return ""
	}
	key = append(key, sum...)
	return base58.Encode(key)
}

// @docs https://learnmeabitcoin.com/technical/public-key-hash
// @docs https://learnmeabitcoin.com/technical/address
func P2PKH(pubKey []byte) string {
	h, _ := hash160(pubKey)

	prefix := []byte{0x00}
	preData := append(prefix, h...)
	sum, _ := checksum(preData)
	addr := append(preData, sum...)

	return base58.Encode(addr)
}

func P2SH(pubKey []byte) string {
	h, _ := hash160(pubKey)

	prefix := []byte{0x05}
	preData := append(prefix, h...)
	sum, _ := checksum(preData)
	addr := append(preData, sum...)

	return base58.Encode(addr)
}

func compressPublicKey(x *big.Int, y *big.Int) []byte {
	var key bytes.Buffer

	// Write header; 0x2 for even y value; 0x3 for odd
	key.WriteByte(byte(0x2) + byte(y.Bit(0)))

	// Write X coord; Pad the key so x is aligned with the LSB. Pad size is key length - header size (1) - xBytes size
	xBytes := x.Bytes()
	for i := 0; i < (PublicKeyCompressedLength - 1 - len(xBytes)); i++ {
		key.WriteByte(0x0)
	}
	key.Write(xBytes)

	return key.Bytes()
}

func uncompressedPublicKey(x *big.Int, y *big.Int) []byte {
	var key bytes.Buffer
	key.WriteByte(byte(0x4))
	key.Write(x.Bytes())
	key.Write(y.Bytes())
	return key.Bytes()
}