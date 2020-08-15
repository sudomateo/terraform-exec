package tfinstall

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/hashicorp/go-getter"
	"golang.org/x/crypto/openpgp"
)

func ensureInstallDir(installDir string) (string, error) {
	if installDir == "" {
		return ioutil.TempDir("", "tfexec")
	}

	if _, err := os.Stat(installDir); err != nil {
		return "", fmt.Errorf("could not access directory %s for installing Terraform: %w", installDir, err)
	}

	return installDir, nil
}

func downloadWithVerification(ctx context.Context, tfVersion string, installDir string) (string, error) {
	osName := runtime.GOOS
	archName := runtime.GOARCH

	// setup: ensure we have a place to put our downloaded terraform binary
	tfDir, err := ensureInstallDir(installDir)
	if err != nil {
		return "", err
	}

	// setup: getter client

	// TODO: actually use this header...
	// httpHeader := make(http.Header)
	// httpHeader.Set("User-Agent", "HashiCorp-tfinstall/"+Version)

	httpGetter := &getter.HttpGetter{
		Netrc: true,
	}
	client := getter.Client{
		Ctx: ctx,
		Getters: map[string]getter.Getter{
			"https": httpGetter,
		},
	}
	client.Mode = getter.ClientModeAny

	// firstly, download and verify the signature of the checksum file

	sumsTmpDir, err := ioutil.TempDir("", "tfinstall")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(sumsTmpDir)

	sumsFilename := "terraform_" + tfVersion + "_SHA256SUMS"
	sumsSigFilename := sumsFilename + ".sig"

	sumsUrl := fmt.Sprintf("%s/%s/%s",
		baseUrl, tfVersion, sumsFilename)
	sumsSigUrl := fmt.Sprintf("%s/%s/%s",
		baseUrl, tfVersion, sumsSigFilename)

	client.Src = sumsUrl
	client.Dst = sumsTmpDir
	err = client.Get()
	if err != nil {
		return "", fmt.Errorf("error fetching checksums: %s", err)
	}

	client.Src = sumsSigUrl
	err = client.Get()
	if err != nil {
		return "", fmt.Errorf("error fetching checksums signature: %s", err)
	}

	sumsPath := filepath.Join(sumsTmpDir, sumsFilename)
	sumsSigPath := filepath.Join(sumsTmpDir, sumsSigFilename)

	err = verifySumsSignature(sumsPath, sumsSigPath)
	if err != nil {
		return "", err
	}

	// secondly, download Terraform itself, verifying the checksum
	url := tfUrl(tfVersion, osName, archName)
	client.Src = url
	client.Dst = tfDir
	client.Mode = getter.ClientModeDir
	err = client.Get()
	if err != nil {
		return "", err
	}

	return filepath.Join(tfDir, "terraform"), nil
}

// verifySumsSignature downloads SHA256SUMS and SHA256SUMS.sig and verifies
// the signature using the HashiCorp public key.
func verifySumsSignature(sumsPath, sumsSigPath string) error {
	el, err := openpgp.ReadArmoredKeyRing(strings.NewReader(hashicorpPublicKey))
	if err != nil {
		return err
	}
	data, err := os.Open(sumsPath)
	if err != nil {
		return err
	}
	sig, err := os.Open(sumsSigPath)
	if err != nil {
		return err
	}
	_, err = openpgp.CheckDetachedSignature(el, data, sig)

	return err
}

func tfUrl(tfVersion, osName, archName string) string {
	sumsFilename := "terraform_" + tfVersion + "_SHA256SUMS"
	sumsUrl := fmt.Sprintf("%s/%s/%s",
		baseUrl, tfVersion, sumsFilename)
	return fmt.Sprintf(
		"%s/%s/terraform_%s_%s_%s.zip?checksum=file:%s",
		baseUrl, tfVersion, tfVersion, osName, archName, sumsUrl,
	)
}
