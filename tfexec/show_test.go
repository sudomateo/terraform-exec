package tfexec

import (
	"context"
	"os"
	"testing"

	"github.com/hashicorp/terraform-exec/tfexec/internal/testutil"
)

func TestShowCmd(t *testing.T) {
	td := testTempDir(t)
	defer os.RemoveAll(td)

	tf, err := NewTerraform(td, tfVersion(t, testutil.Latest012))
	if err != nil {
		t.Fatal(err)
	}

	// empty env, to avoid environ mismatch in testing
	tf.SetEnv(map[string]string{})

	// defaults
	showCmd := tf.showCmd(context.Background())

	assertCmd(t, []string{
		"show",
		"-json",
		"-no-color",
	}, nil, showCmd)
}

func TestShowStateCmd(t *testing.T) {
	td := testTempDir(t)
	defer os.RemoveAll(td)

	tf, err := NewTerraform(td, tfVersion(t, testutil.Latest012))
	if err != nil {
		t.Fatal(err)
	}

	// empty env, to avoid environ mismatch in testing
	tf.SetEnv(map[string]string{})

	showCmd := tf.showCmd(context.Background(), "statefilepath")

	assertCmd(t, []string{
		"show",
		"-json",
		"-no-color",
		"statefilepath",
	}, nil, showCmd)
}

func TestShowPlanCmd(t *testing.T) {
	td := testTempDir(t)
	defer os.RemoveAll(td)

	tf, err := NewTerraform(td, tfVersion(t, testutil.Latest012))
	if err != nil {
		t.Fatal(err)
	}

	// empty env, to avoid environ mismatch in testing
	tf.SetEnv(map[string]string{})

	showCmd := tf.showCmd(context.Background(), "planfilepath")

	assertCmd(t, []string{
		"show",
		"-json",
		"-no-color",
		"planfilepath",
	}, nil, showCmd)
}
