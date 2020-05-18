package utils

import (
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// Sha256sum calulates sha256 hash of file
func Sha256sum(file string) string {
	input, err := os.Open(file)
	if err != nil {
		panic(err)
	}
	defer input.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, input); err != nil {
		log.Fatal(err)
	}
	sum := hash.Sum(nil)

	return fmt.Sprintf("%x", sum)
}

//FileNameWithoutExtension basename of file
func FileNameWithoutExtension(fileName string) string {
	return strings.TrimSuffix(fileName, filepath.Ext(fileName))
}
