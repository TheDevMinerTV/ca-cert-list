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
	"strings"

	"html/template"
	"os"
	"path/filepath"

	_ "embed"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

//go:embed templates/index.gohtml
var IndexTemplate string

type Certificate struct {
	Name string `yaml:"name"`
	Path string `yaml:"path"`
	/// FileName is the filename that this file will be downloaded as
	FileName    *string `yaml:"filename"`
	Description *string `yaml:"description"`
	Deprecated  bool    `yaml:"deprecated"`
}

type CertificateGroup struct {
	Name         string        `yaml:"name"`
	Description  *string       `yaml:"description"`
	Deprecated   bool          `yaml:"deprecated"`
	Certificates []Certificate `yaml:"certificates"`
}

type Config struct {
	Groups []CertificateGroup `yaml:"groups"`
}

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	output := flag.String("output", "index.html", "output file")
	configPath := flag.String("config", "config.yml", "config file")

	flag.Parse()

	if *configPath == "" {
		log.Fatal().Msg("--config is required")
	}

	config, err := parseConfig(*configPath)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to parse config")
	}
	log.Debug().Msg("loaded configuration")

	index, err := template.New("index").Parse(IndexTemplate)
	if err != nil {
		log.Fatal().Err(err).Msg("error parsing template")
	}
	log.Debug().Msg("parsed template")

	if err := os.MkdirAll(*output, 0755); err != nil {
		log.Fatal().Err(err).Msg("error creating output folder")
	}

	groupComponents := make([]GroupComponentParams, 0)
	for _, group := range config.Groups {
		components := make([]CertificateComponentParams, 0)
		for _, certificate := range group.Certificates {
			component, err := processCertificate(certificate, group.Name, filepath.Join(*output, "certificates", group.Name))
			if err != nil {
				log.Error().Err(err).Msgf("failed to process %s", certificate.Name)
				continue
			}

			components = append(components, *component)
		}

		groupComponents = append(groupComponents, GroupComponentParams{
			Name:         group.Name,
			Description:  group.Description,
			Deprecated:   group.Deprecated,
			Certificates: components,
		})
	}

	f, err := os.Create(filepath.Join(*output, "index.html"))
	if err != nil {
		log.Fatal().Err(err).Msg("error creating index.html file")
	}
	defer f.Close()

	if err := index.Execute(f, groupComponents); err != nil {
		log.Fatal().Err(err).Msg("error rendering index.html")
	}
}

type GroupComponentParams struct {
	Name         string
	Description  *string
	Deprecated   bool
	Certificates []CertificateComponentParams
}

type CertificateComponentParams struct {
	Name            string
	Description     *string
	Deprecated      bool
	File            string
	Expiry          string
	ExpiryTimestamp int64
	SHA256          string
	SHA1            string
	MD5             string
}

func parseCertificate(f *os.File) (*x509.Certificate, error) {
	rawCertificate, err := io.ReadAll(f)
	if err != nil {
		return nil, errors.Wrap(err, "reading certificate file")
	}
	if _, err := f.Seek(0, 0); err != nil {
		return nil, errors.Wrap(err, "seeking certificate file")
	}

	block, rest := pem.Decode(rawCertificate)
	if len(rest) > 0 {
		log.Warn().Msg("certificate contains extra data")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, errors.Wrap(err, "parsing certificate")
	}

	return cert, nil
}

func hashFile(f *os.File, hash func() hash.Hash) (string, error) {
	if _, err := f.Seek(0, 0); err != nil {
		return "", errors.Wrap(err, "seeking certificate file")
	}

	h := hash()
	if _, err := io.Copy(h, f); err != nil {
		return "", errors.Wrap(err, "hashing certificate")
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func copyFile(f *os.File, dst string) error {
	if _, err := f.Seek(0, 0); err != nil {
		return errors.Wrap(err, "seeking certificate file")
	}

	dstFile, err := os.Create(dst)
	if err != nil {
		return errors.Wrap(err, "creating certificate file")
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, f); err != nil {
		return errors.Wrap(err, "copying certificate file")
	}

	return nil
}

func parseConfig(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	config := &Config{}
	if err := yaml.NewDecoder(f).Decode(config); err != nil {
		return nil, err
	}

	return config, nil
}

func processCertificate(spec Certificate, groupName, output string) (*CertificateComponentParams, error) {
	log.Debug().Str("name", spec.Name).Msg("processing certificate")

	f, err := os.Open(spec.Path)
	if err != nil {
		return nil, errors.Wrap(err, "opening certificate file")
	}
	defer f.Close()

	cert, err := parseCertificate(f)
	if err != nil {
		return nil, errors.Wrap(err, "parsing certificate")
	}

	expiry := cert.NotAfter.Format("2006-01-02 15:04:05")
	expiryTimestamp := cert.NotAfter.Unix()
	log.Trace().Str("expiry", expiry).Int64("expiryTimestamp", expiryTimestamp).Msg("read expiry")

	sha256, err := hashFile(f, sha256.New)
	if err != nil {
		return nil, errors.Wrap(err, "hashing with SHA256")
	}
	log.Trace().Str("hash", sha256).Msg("hashed with SHA256")

	sha1, err := hashFile(f, sha1.New)
	if err != nil {
		return nil, errors.Wrap(err, "hashing with SHA1")
	}
	log.Trace().Str("hash", sha1).Msg("hashed with sha1")

	md5, err := hashFile(f, md5.New)
	if err != nil {
		return nil, errors.Wrap(err, "hashing with MD5")
	}
	log.Trace().Str("hash", md5).Msg("hashed with MD5")

	if err := os.MkdirAll(output, 0755); err != nil {
		return nil, errors.Wrap(err, "creating folder structure")
	}
	log.Debug().Msg("created folder structure")

	fileName := getFileName(spec) + ".crt"

	if err := copyFile(f, filepath.Join(output, fileName)); err != nil {
		return nil, errors.Wrap(err, "copying certificate file")
	}
	log.Trace().Msg("copied certificate file")

	return &CertificateComponentParams{
		Name:            spec.Name,
		Description:     spec.Description,
		Deprecated:      spec.Deprecated,
		File:            filepath.Join("/certificates/", groupName, fileName),
		Expiry:          expiry,
		ExpiryTimestamp: expiryTimestamp,
		SHA256:          sha256,
		SHA1:            sha1,
		MD5:             md5,
	}, nil
}

func getFileName(cert Certificate) string {
	if cert.FileName != nil {
		return *cert.FileName
	}

	return strings.ReplaceAll(strings.ReplaceAll(cert.Name, " ", "_"), ".", "_")
}
