package e2etest

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/go-version"

	"github.com/hashicorp/terraform-exec/tfexec"
	"github.com/hashicorp/terraform-exec/tfexec/internal/testutil"
)

const testFixtureDir = "testdata"
const masterRef = "refs/heads/master"

func runTest(t *testing.T, fixtureName string, cb func(t *testing.T, tfVersion *version.Version, tf *tfexec.Terraform)) {
	t.Helper()
	runTestVersions(t, []string{
		testutil.Latest011,
		testutil.Latest012,
		testutil.Latest013,
		masterRef,
	}, fixtureName, cb)
}

// runTestVersions should probably not be used directly, better to use
// t.Skip in your test with a comment as to why you shouldn't test on a version
func runTestVersions(t *testing.T, versions []string, fixtureName string, cb func(t *testing.T, tfVersion *version.Version, tf *tfexec.Terraform)) {
	t.Helper()

	alreadyRunVersions := map[string]bool{}
	for _, tfv := range versions {
		t.Run(fmt.Sprintf("%s-%s", fixtureName, tfv), func(t *testing.T) {
			if alreadyRunVersions[tfv] {
				t.Skipf("already run version %q", tfv)
			}
			alreadyRunVersions[tfv] = true

			td, err := ioutil.TempDir("", "tf")
			if err != nil {
				t.Fatalf("error creating temporary test directory: %s", err)
			}
			t.Cleanup(func() {
				os.RemoveAll(td)
			})

			// TODO: do this in a cleaner way than string comparison?
			var execPath string
			switch {
			case strings.HasPrefix(tfv, "refs/"):
				execPath = tfcache.GitRef(t, tfv)
			default:
				execPath = tfcache.Version(t, tfv)
			}

			tf, err := tfexec.NewTerraform(td, execPath)
			if err != nil {
				t.Fatal(err)
			}

			if fixtureName != "" {
				err = copyFiles(filepath.Join(testFixtureDir, fixtureName), td)
				if err != nil {
					t.Fatalf("error copying config file into test dir: %s", err)
				}
			}

			var stdouterr bytes.Buffer
			tf.SetStdout(&stdouterr)
			tf.SetStderr(&stdouterr)

			// TODO: capture panics here?
			cb(t, version.Must(version.NewVersion(tfv)), tf)

			t.Logf("CLI Output:\n%s", stdouterr.String())
		})
	}
}

func copyFiles(path string, dstPath string) error {
	infos, err := ioutil.ReadDir(path)
	if err != nil {
		return err
	}

	for _, info := range infos {
		srcPath := filepath.Join(path, info.Name())
		if info.IsDir() {
			newDir := filepath.Join(dstPath, info.Name())
			err = os.MkdirAll(newDir, info.Mode())
			if err != nil {
				return err
			}
			err = copyFiles(srcPath, newDir)
			if err != nil {
				return err
			}
		} else {
			err = copyFile(srcPath, dstPath)
			if err != nil {
				return err
			}
		}

	}
	return nil
}

func copyFile(path string, dstPath string) error {
	srcF, err := os.Open(path)
	if err != nil {
		return err
	}
	defer srcF.Close()

	di, err := os.Stat(dstPath)
	if err != nil {
		return err
	}
	if di.IsDir() {
		_, file := filepath.Split(path)
		dstPath = filepath.Join(dstPath, file)
	}

	dstF, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer dstF.Close()

	if _, err := io.Copy(dstF, srcF); err != nil {
		return err
	}

	return nil
}

// filesEqual returns true iff the two files have the same contents.
func filesEqual(file1, file2 string) (bool, error) {
	sf, err := os.Open(file1)
	if err != nil {
		return false, err
	}

	df, err := os.Open(file2)
	if err != nil {
		return false, err
	}

	sscan := bufio.NewScanner(sf)
	dscan := bufio.NewScanner(df)

	for sscan.Scan() {
		dscan.Scan()
		if !bytes.Equal(sscan.Bytes(), dscan.Bytes()) {
			return true, nil
		}
	}

	return false, nil
}
