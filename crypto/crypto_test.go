package crypto

import (
	"bytes"
	"testing"
)

var (
	key  = []byte{218, 20, 23, 3, 170, 139, 80, 118, 210, 111, 134, 203, 80, 199, 209, 112, 166, 114, 128, 115, 232, 149, 147, 64, 95, 70, 25, 87, 137, 77, 71, 105}
	data = []byte("hello,world")
)

func TestCurve25519ECDH_GenerateSharedSecret(t *testing.T) {
	ecdh := NewCurve25519ECDH()
	privKeyA, pubKeyA, err := ecdh.GenerateKey()
	if err != nil {
		t.Fatal("Fail to Generate Key")
	}

	privKeyB, pubKeyB, err := ecdh.GenerateKey()
	if err != nil {
		t.Fatal("Fail to Generate Key")
	}

	pubKeyAMarshal := ecdh.Marshal(pubKeyA)
	pubKeyBMarshal := ecdh.Marshal(pubKeyB)

	var ok bool
	pubKeyA, ok = ecdh.Unmarshal(pubKeyAMarshal)
	if !ok {
		t.Fatal("Fail to Unmarshal public key")
	}

	pubKeyB, ok = ecdh.Unmarshal(pubKeyBMarshal)
	if !ok {
		t.Fatal("Fail to Unmarshal public key")
	}

	secretA, err := ecdh.GenerateSharedSecret(privKeyA, pubKeyB)
	if err != nil {
		t.Fatal("Fail to GenerateSharedSecret")
	}
	t.Log(secretA)

	secretB, err := ecdh.GenerateSharedSecret(privKeyB, pubKeyA)
	if err != nil {
		t.Fatal("Fail to GenerateSharedSecret")
	}
	t.Log(secretB)

	if !bytes.Equal(secretA, secretB) {
		t.Fatal("Fail to generate equal secret")
	}
}

func TestEncrypt(t *testing.T) {
	e, err := Encrypt(data, key)
	if err != nil {
		t.Fatal("Fail to encrypt data: ", err)
	}

	t.Log(e)

	d, err := Decrypt(e, key)
	if err != nil {
		t.Fatal("Fail to decrypt data: ", err)
	}

	if !bytes.Equal(d, data) {
		t.Fatal("Fail to decrypt data")
	}
}
