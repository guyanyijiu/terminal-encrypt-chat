package crypto

import (
	"crypto"
	"crypto/rand"
	"errors"
	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/curve25519"
)

type ECDH interface {
	GenerateKey() (crypto.PrivateKey, crypto.PublicKey, error)
	Marshal(crypto.PublicKey) []byte
	Unmarshal([]byte) (crypto.PublicKey, bool)
	GenerateSharedSecret(crypto.PrivateKey, crypto.PublicKey) ([]byte, error)
}

type curve25519ECDH struct {
	ECDH
}

func NewCurve25519ECDH() ECDH {
	return &curve25519ECDH{}
}

func (e *curve25519ECDH) GenerateKey() (crypto.PrivateKey, crypto.PublicKey, error) {
	var publicKey, privateKey [32]byte
	_, err := rand.Read(privateKey[:])
	if err != nil {
		return nil, nil, err
	}
	privateKey[0] &= 248
	privateKey[31] &= 127
	privateKey[31] |= 64
	curve25519.ScalarBaseMult(&publicKey, &privateKey)
	return &privateKey, &publicKey, nil
}

func (e *curve25519ECDH) Marshal(p crypto.PublicKey) []byte {
	publicKey := p.(*[32]byte)
	return publicKey[:]
}

func (e *curve25519ECDH) Unmarshal(data []byte) (crypto.PublicKey, bool) {
	var publicKey [32]byte
	if len(data) != 32 {
		return nil, false
	}
	copy(publicKey[:], data)
	return &publicKey, true
}

func (e *curve25519ECDH) GenerateSharedSecret(privateKey crypto.PrivateKey, publicKey crypto.PublicKey) ([]byte, error) {
	var private, public, secret *[32]byte
	private = privateKey.(*[32]byte)
	public = publicKey.(*[32]byte)
	secret = new([32]byte)
	curve25519.ScalarMult(secret, private, public)
	return secret[:], nil
}

func Encrypt(data []byte, key []byte) ([]byte, error) {
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, chacha20poly1305.NonceSizeX)
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	encrypted := aead.Seal(nonce[:], nonce, data, nil)

	return encrypted, nil
}

func Decrypt(encrypted []byte, key []byte) ([]byte, error) {
	if len(encrypted) < chacha20poly1305.NonceSizeX {
		return nil, errors.New("Invalid encrypted data")
	}
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, err
	}

	nonce := encrypted[:chacha20poly1305.NonceSizeX]
	return aead.Open(nil, nonce, encrypted[chacha20poly1305.NonceSizeX:], nil)
}
