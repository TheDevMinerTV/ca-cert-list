package main

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"hash"
	"io"

	"html/template"
	"os"
	"path/filepath"

	_ "embed"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

//go:embed templates/index.gohtml
var IndexTemplate string

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	output := flag.String("output", "index.html", "output file")
	root := flag.String("root", "./certificates", "root folder")

	flag.Parse()

	certificateNames := flag.Args()

	index, err := template.New("index").Parse(IndexTemplate)
	if err != nil {
		log.Fatal().Err(err).Msg("error parsing template")
	}
	log.Debug().Msg("parsed template")

	if err := os.MkdirAll(*output, 0755); err != nil {
		log.Fatal().Err(err).Msg("error creating output folder")
	}

	if err := os.MkdirAll(filepath.Join(*output, "certificates"), 0755); err != nil {
		log.Fatal().Err(err).Msg("error creating certificates folder")
	}
	log.Debug().Msg("created folder structure")

	components := make([]ComponentParams, 0, len(certificateNames))
	for _, certificateName := range certificateNames {
		log.Debug().Str("certificate", certificateName).Msg("processing certificate")
		folder := filepath.Join(*root, certificateName)
		certificateFile := filepath.Join(folder, "certificate.crt")

		description, err := os.ReadFile(filepath.Join(folder, "description.txt"))
		if err != nil {
			log.Error().Err(err).Msg("error reading description")
			continue
		}
		log.Trace().Str("description", string(description)).Msg("read description")

		f, err := os.Open(certificateFile)
		if err != nil {
			log.Error().Err(err).Msg("error opening certificate file")
			continue
		}
		defer f.Close()

		cert, err := parseCertificate(f)
		if err != nil {
			log.Error().Err(err).Msg("error parsing certificate")
			continue
		}

		expiry := cert.NotAfter.Format("2006-01-02 15:04:05")
		expiryTimestamp := cert.NotAfter.Unix()
		log.Trace().Str("expiry", expiry).Int64("expiryTimestamp", expiryTimestamp).Msg("read expiry")

		sha256, err := hashFile(f, sha256.New)
		if err != nil {
			log.Error().Err(err).Msg("error hashing certificate with SHA256")
			continue
		}
		log.Trace().Str("sha256", sha256).Msg("hashed with sha256")

		sha1, err := hashFile(f, sha1.New)
		if err != nil {
			log.Error().Err(err).Msg("error hashing certificate with SHA1")
			continue
		}
		log.Trace().Str("sha1", sha1).Msg("hashed with sha1")

		md5, err := hashFile(f, md5.New)
		if err != nil {
			log.Error().Err(err).Msg("error hashing certificate with MD5")
			continue
		}
		log.Trace().Str("md5", md5).Msg("hashed with md5")

		components = append(components, ComponentParams{
			Name:            certificateName,
			Description:     string(description),
			Expiry:          expiry,
			ExpiryTimestamp: expiryTimestamp,
			SHA256:          sha256,
			SHA1:            sha1,
			MD5:             md5,
		})

		if err := copyFile(f, filepath.Join(*output, "certificates", certificateName+".crt")); err != nil {
			log.Error().Err(err).Msg("error copying certificate file")
			continue
		}
		log.Trace().Msg("copied certificate file")
	}

	f, err := os.Create(filepath.Join(*output, "index.html"))
	if err != nil {
		log.Fatal().Err(err).Msg("error creating index.html file")
	}
	defer f.Close()

	if err := index.Execute(f, components); err != nil {
		log.Fatal().Err(err).Msg("error rendering index.html")
	}
}

type ComponentParams struct {
	Name            string
	Description     string
	Expiry          string
	ExpiryTimestamp int64
	SHA256          string
	SHA1            string
	MD5             string
}

func parseCertificate(f *os.File) (*x509.Certificate, error) {
	rawCertificate, err := io.ReadAll(f)
	if err != nil {
		return nil, errors.Wrap(err, "error reading certificate file")
	}
	if _, err := f.Seek(0, 0); err != nil {
		return nil, errors.Wrap(err, "error seeking certificate file")
	}

	block, rest := pem.Decode(rawCertificate)
	if len(rest) > 0 {
		log.Warn().Msg("certificate contains extra data")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, errors.Wrap(err, "error parsing certificate")
	}

	return cert, nil
}

func hashFile(f *os.File, hash func() hash.Hash) (string, error) {
	if _, err := f.Seek(0, 0); err != nil {
		return "", errors.Wrap(err, "error seeking certificate file")
	}

	h := hash()
	if _, err := io.Copy(h, f); err != nil {
		return "", errors.Wrap(err, "error hashing certificate")
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func copyFile(f *os.File, dst string) error {
	if _, err := f.Seek(0, 0); err != nil {
		return errors.Wrap(err, "error seeking certificate file")
	}

	dstFile, err := os.Create(dst)
	if err != nil {
		return errors.Wrap(err, "error creating certificate file")
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, f); err != nil {
		return errors.Wrap(err, "error copying certificate file")
	}

	return nil
}
