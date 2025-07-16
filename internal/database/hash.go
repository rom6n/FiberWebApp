package database

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Error custom types:

type MTTError struct{}
type DecodeError struct{}

func (MTTError) Error() string {
	return "Memory, time, threads values are damaged"
}
func (DecodeError) Error() string {
	return "Error while decoding bytes"
}

//-------------------

func HashString(str string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	hash := argon2.IDKey(
		[]byte(str),
		salt,
		1,
		64*1024,
		4,
		32,
	)
	hashString := base64.RawStdEncoding.EncodeToString(hash)
	saltString := base64.RawStdEncoding.EncodeToString(salt)

	return fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		64*1024, 1, 4,
		saltString,
		hashString,
	), nil

}

func VerifyHash(str string, hashString string) (bool, error) {
	info := strings.Split(hashString, "$")

	var memory, time uint32
	var threads uint8
	_, MTTErr := fmt.Sscanf(info[3], "m=%d,t=%d,p=%d", &memory, &time, &threads)
	if MTTErr != nil {
		return false, MTTError{}
	}

	salt, err := base64.RawStdEncoding.DecodeString(info[4])
	if err != nil {
		return false, DecodeError{}
	}

	hash, err := base64.RawStdEncoding.DecodeString(info[5])
	if err != nil {
		return false, DecodeError{}
	}

	hashToCompare := argon2.IDKey(
		[]byte(str),
		salt,
		time,
		memory,
		threads,
		32,
	)

	return subtle.ConstantTimeCompare(hashToCompare, hash) == 1, nil
}
