// Package info gerencia controle e configuração modular, com suporte a arquivos separados por módulo.
package info

import (
	"encoding/json"
	"os"
	"path/filepath"

	gl "github.com/kubex-ecosystem/logz"
)

type Reference struct {
	Name            string `json:"name"`
	ApplicationName string `json:"application"`
	Bin             string `json:"bin"`
	Version         string `json:"version"`
}

// Control representa a configuração de controle de um módulo.
type Control struct {
	Reference     Reference `json:"-"` // Usado internamente para nome do arquivo, nunca exportado
	SchemaVersion int       `json:"schema_version"`
	IPC           IPC       `json:"ipc"`
	Bitreg        Bitreg    `json:"bitreg"`
	KV            KV        `json:"kv"`
	Seq           int       `json:"seq"`
	EpochNS       int64     `json:"epoch_ns"`
}

func (c *Control) GetName() string    { return c.Reference.Name }
func (c *Control) GetVersion() string { return c.Reference.Version }

// LoadControlByModule carrega o controle de um arquivo específico do módulo.
func LoadControlByModule(dir string, moduleName string) (*Control, error) {
	file := filepath.Join(dir, gl.Sprintf("control_%s.json", moduleName))
	f, err := os.Open(file)
	if err != nil {
		return nil, gl.Errorf("erro ao abrir %s: %w", file, err)
	}
	defer f.Close()
	var c Control
	dec := json.NewDecoder(f)
	if err := dec.Decode(&c); err != nil {
		return nil, gl.Errorf("erro ao decodificar %s: %w", file, err)
	}
	c.Reference = Reference{Name: moduleName}
	return &c, nil
}

// SaveControl salva o controle do módulo em arquivo separado.
func (c *Control) SaveControl(dir string) error {
	if c.Reference.Name == "" {
		return gl.Errorf("Reference.Name não pode ser vazio para salvar o controle")
	}
	file := filepath.Join(dir, gl.Sprintf("control_%s.json", c.Reference.Name))
	f, err := os.Create(file)
	if err != nil {
		return gl.Errorf("erro ao criar %s: %w", file, err)
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	// Reference não é exportado
	return enc.Encode(c)
}
