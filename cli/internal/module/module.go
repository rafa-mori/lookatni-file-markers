// Package module provides internal types and functions for the Grompt application.
package module

import (
	gl "github.com/kubex-ecosystem/logz"
	cc "github.com/kubex-ecosystem/lookatni-file-markers/cmd/cli"
	vs "github.com/kubex-ecosystem/lookatni-file-markers/internal/module/version"
	"github.com/spf13/cobra"

	"os"
	"strings"
)

type LookAtni struct {
	parentCmdName string
	printBanner   bool
}

func (m *LookAtni) Alias() string { return "" }
func (m *LookAtni) ShortDescription() string {
	return "LookAtni: A meta-framework for handle with your files like never before."
}
func (m *LookAtni) LongDescription() string {
	return `LookAtni: A meta-framework for handling your files like never before.
	It provides a command-line interface to extract, validate, and generate file markers for composing and decomposing entire projects, and much more!`
}
func (m *LookAtni) Usage() string {
	return "lookatni [command] [args]"
}
func (m *LookAtni) Examples() []string {
	return []string{
		"lookatni extract <file> <output-dir>",
		"lookatni validate <file>",
		"lookatni generate <source-dir> <output-file>",
	}
}
func (m *LookAtni) Active() bool {
	return true
}
func (m *LookAtni) Module() string {
	return "lookatni"
}
func (m *LookAtni) Execute() error { return m.Command().Execute() }
func (m *LookAtni) Command() *cobra.Command {
	gl.Log("debug", "Starting LookAtni CLI...")

	var rtCmd = &cobra.Command{
		Use:     m.Module(),
		Aliases: []string{m.Alias()},
		Example: m.concatenateExamples(),
		Version: vs.GetVersion(),
		Annotations: cc.GetDescriptions([]string{
			m.LongDescription(),
			m.ShortDescription(),
		}, m.printBanner),
	}

	rtCmd.AddCommand(cc.ServiceCmdList()...)
	rtCmd.AddCommand(cc.KubingCmd())
	rtCmd.AddCommand(vs.CliCommand())

	// Set usage definitions for the command and its subcommands
	setUsageDefinition(rtCmd)
	for _, c := range rtCmd.Commands() {
		setUsageDefinition(c)
		if !strings.Contains(strings.Join(os.Args, " "), c.Use) {
			if c.Short == "" {
				c.Short = c.Annotations["description"]
			}
		}
	}

	return rtCmd
}
func (m *LookAtni) SetParentCmdName(rtCmd string) {
	m.parentCmdName = rtCmd
}
func (m *LookAtni) concatenateExamples() string {
	examples := ""
	rtCmd := m.parentCmdName
	if rtCmd != "" {
		rtCmd = rtCmd + " "
	}
	for _, example := range m.Examples() {
		examples += rtCmd + example + "\n  "
	}
	return examples
}
