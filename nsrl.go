package filetrove

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/schollz/progressbar/v3"
	"go.etcd.io/bbolt"
)

func CreateNSRLBoltDB(nsrlsourcefile string, nsrlversion string, nsrldbfile string) error {
	db, err := bbolt.Open(nsrldbfile, 0600, nil)
	if err != nil {
		return err
	}
	defer db.Close()

	file, err := os.Open(nsrlsourcefile)
	if err != nil {
		return err
	}
	defer file.Close()

	batchSize := 100000
	values := make([]string, 0, batchSize)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		hash := scanner.Text()
		values = append(values, hash)

		if len(values) == batchSize {
			err := db.Update(func(tx *bbolt.Tx) error {
				bucket, err := tx.CreateBucketIfNotExists([]byte("sha1"))
				if err != nil {
					return err
				}
				// Reduce file size
				bucket.FillPercent = 0.9

				for _, value := range values {
					err := bucket.Put([]byte(strings.ToLower(value)), []byte("true"))
					if err != nil {
						return err
					}
				}
				return nil
			})

			if err != nil {
				return err
			}
			values = values[:0]
		}

	}

	if len(values) > 0 {
		err := db.Update(func(tx *bbolt.Tx) error {
			bucket, err := tx.CreateBucketIfNotExists([]byte("sha1"))
			if err != nil {
				return err
			}

			for _, value := range values {
				err := bucket.Put([]byte(strings.ToLower(value)), []byte("true"))
				if err != nil {
					return err
				}
			}

			return nil
		})
		if err != nil {
			return err
		}
	}
	// After the last sha1 was put into the boltdb
	// we add the key nsrlversion with the value provided via flag
	err = db.Update(func(tx *bbolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte("sha1"))
		if err != nil {
			return err
		}
		err = bucket.Put([]byte("nsrlversion"), []byte(nsrlversion))
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

// GetNSRL downloads a prepared BoltDB database file from an online storage
func GetNSRL(install string) error {
	req, err := http.NewRequest("GET", "https://download.fritz.wtf/nsrl.db.gz", nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return errors.New("Could not download NSRL database. Server returned: " + resp.Status)
	}

	f, err := os.OpenFile(filepath.Join(install, "db", "nsrl.db.gz"), os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	bar := progressbar.DefaultBytes(
		resp.ContentLength,
		"downloading",
	)
	io.Copy(io.MultiWriter(f, bar), resp.Body)

	return nil
}

// UnzipNSRL unzips the nsrl.db.gz file and returns an error if it fails
func UnzipNSRL(nsrlZipFile string, outputDir string) error {
	// Open the gzip file for reading
	gzipFile, err := os.Open(nsrlZipFile)
	if err != nil {
		return errors.New("Could not open nsrl.db.gz file: " + err.Error())
	}
	defer gzipFile.Close()

	// Create the corresponding output file
	outputFile, err := os.Create(filepath.Join(outputDir, "nsrl.db"))
	if err != nil {
		return errors.New("Could not create output file: " + err.Error())
	}
	defer outputFile.Close()

	// Create a gzip reader
	gzipReader, err := gzip.NewReader(gzipFile)
	if err != nil {
		return errors.New("Could not create gzip reader:" + err.Error())
	}
	defer gzipReader.Close()

	// Set up progress bar
	bar := progressbar.DefaultBytes(
		-1,
		"Uncompressing NSRL database",
	)
	// Copy the contents of the gzip file to the output file
	_, err = io.Copy(io.MultiWriter(outputFile, bar), gzipReader)
	if err != nil {
		return errors.New("Could not copy gzip content: " + err.Error())
	}
	return err
}

// ChecksumNSRL checks a NSRL BoltDB's checksum that is provided with a sidecar file
func ChecksumNSRL(nsrldbfile string) {
	Hashit(nsrldbfile, "blake2b-512")
}

// ConnectNSRL connects to local bbolt NSRL file
func ConnectNSRL(nsrldbfile string) (*bbolt.DB, error) {
	db, err := bbolt.Open(nsrldbfile, 0600, nil)
	if err != nil {
		return nil, err
	}

	return db, nil
}

// GetValueNSRL reads bbolt database and checks if a given sha1 hash is present in the database
func GetValueNSRL(db *bbolt.DB, sha1hash []byte) (bool, error) {
	var fileIsInNSRL bool

	err := db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("sha1"))
		if b == nil {
			return errors.New("Could not connect to bucket.")
		}

		// the byte array translates to UTF-8 "true"
		fileIsInNSRL = bytes.Equal(b.Get(sha1hash), []byte{116, 114, 117, 101})
		// return nil to complete the transaction
		return nil
	})
	return fileIsInNSRL, err
}

// GetNSRLVersion from BoltDB
func GetNSRLVersion(db *bbolt.DB) (string, error) {
	var nsrlVersion string

	err := db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("sha1"))
		if b == nil {
			return errors.New("Could not connect to bucket.")
		}

		// the byte array translates to UTF-8 "true"
		nsrlVersion = string(b.Get([]byte("nsrlversion")))
		// return nil to complete the transaction
		return nil
	})
	return nsrlVersion, err
}
