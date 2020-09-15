package clusterloader2

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/octago/sflags/gen/gpflag"
	"k8s.io/klog"

	"sigs.k8s.io/kubetest2/pkg/exec"
	suite "sigs.k8s.io/kubetest2/pkg/testers/clusterloader2/suite"
)

type Tester struct {
	Suites        string `desc:"Comma separated list of standard scale testing suites e.g. load, density"`
	TestOverrides string `desc:"Comma separated list of paths to the config override files. The latter overrides take precedence over changes in former files."`
	TestConfigs   string `desc:"Comma separated list of paths to test config files."`
	Provider      string `desc:"The type of cluster provider used (e.g gke, gce, skeleton)"`
	KubeConfig    string `desc:"Path to kubeconfig"`
	RepoRoot      string `desc:"Path to repository root of kubernetes/perf-tests"`
	Nodes         int    `desc:"Number of nodes in the cluster. 0 will auto-detect schedulable nodes."`
}

func NewDefaultTester() *Tester {
	return &Tester{
		// TODO(amwat): pass kubetest2 deployer info here if possible
		Provider:   "skeleton",
		KubeConfig: os.Getenv("KUBECONFIG"),
	}
}

// Test runs the test
func (t *Tester) Test() error {
	if t.RepoRoot == "" {
		return fmt.Errorf("required path to kubernetes/perf-tests repository")
	}

	var testConfigs, testOverrides []string
	testConfigs = append(testConfigs, strings.Split(t.TestConfigs, ",")...)
	testOverrides = append(testOverrides, strings.Split(t.TestOverrides, ",")...)

	sweets := strings.Split(t.Suites, ",")
	for _, sweet := range sweets {
		if s := suite.GetSuite(sweet); s != nil {
			if s.TestConfigs != nil && len(s.TestConfigs) > 0 {
				testConfigs = append(testConfigs, s.TestConfigs...)
			}
			if s.TestOverrides != nil && len(s.TestOverrides) > 0 {
				testOverrides = append(testOverrides, s.TestOverrides...)
			}
		}
	}

	cmdArgs := []string{
		"run",
		"cmd/clusterloader.go",
	}

	args := []string{
		"--provider=" + t.Provider,
		"--kubeconfig=" + t.KubeConfig,
		"--report-dir=" + filepath.Join(os.Getenv("ARTIFACTS"), "clusterloader2"),
	}
	for _, tc := range testConfigs {
		if tc != "" {
			args = append(args, "--testconfig="+tc)
		}
	}
	for _, to := range testOverrides {
		if to != "" {
			args = append(args, "--testoverrides="+to)
		}
	}

	// TODO(amwat): get prebuilt binaries
	cmd := exec.Command("go", append(cmdArgs, args...)...)
	exec.InheritOutput(cmd)
	cmd.SetDir(filepath.Join(t.RepoRoot, "clusterloader2"))
	klog.V(2).Infof("running clusterloader2 %s", args)
	return cmd.Run()
}

func (t *Tester) Execute() error {
	fs, err := gpflag.Parse(t)
	if err != nil {
		return fmt.Errorf("failed to initialize tester: %v", err)
	}

	klog.InitFlags(nil)
	fs.AddGoFlagSet(flag.CommandLine)

	help := fs.BoolP("help", "h", false, "")
	if err := fs.Parse(os.Args); err != nil {
		return fmt.Errorf("failed to parse flags: %v", err)
	}

	if *help {
		fs.SetOutput(os.Stdout)
		fs.PrintDefaults()
		return nil
	}

	return t.Test()
}

func Main() {
	t := NewDefaultTester()
	if err := t.Execute(); err != nil {
		klog.Fatalf("failed to run clusterloader2 tester: %v", err)
	}
}
