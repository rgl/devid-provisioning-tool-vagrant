package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"

	"github.com/google/go-attestation/attest"
	"github.com/olekukonko/tablewriter"
)

type table struct {
	header []string
	data   [][]string
}

func (t *table) Render() {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetAutoWrapText(false)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetRowLine(true)
	table.SetHeader(t.header)
	table.AppendBulk(t.data)
	table.Render()
}

func getCertificateText(der []byte) (string, error) {
	cmd := exec.Command("openssl", "x509", "-text", "-inform", "der")
	cmd.Stdin = bytes.NewReader(der)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

func getCertificateFileText(path string) (string, error) {
	cmd := exec.Command("openssl", "x509", "-text", "-in", path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

func getTPMInfo() (*table, error) {
	tpm, err := attest.OpenTPM(&attest.OpenConfig{
		TPMVersion: attest.TPMVersion20,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open the TPM: %v", err)
	}
	defer tpm.Close()

	info, err := tpm.Info()
	if err != nil {
		return nil, fmt.Errorf("failed to get information from TPM: %v", err)
	}

	return &table{
		header: []string{"Property", "Value"},
		data: [][]string{
			{"Vendor", info.VendorInfo},
			{"Manufacturer", info.Manufacturer.String()},
		},
	}, nil
}

func getTPMEndorsementKeys() (*table, error) {
	tpm, err := attest.OpenTPM(&attest.OpenConfig{
		TPMVersion: attest.TPMVersion20,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open the TPM: %v", err)
	}
	defer tpm.Close()

	eks, err := tpm.EKs()
	if err != nil {
		return nil, fmt.Errorf("failed to get EKs from TPM: %v", err)
	}

	data := make([][]string, 0)
	for i, ek := range eks {
		name := fmt.Sprintf("#%d", i)
		value, err := getCertificateText(ek.Certificate.Raw)
		if err != nil {
			value = fmt.Sprintf("ERROR: %v", err)
		}
		data = append(data, []string{name, string(value)})
	}

	return &table{
		header: []string{"EK", "Content"},
		data:   data,
	}, nil
}

func getSwtpmCertificates() (*table, error) {
	data := make([][]string, 0)

	paths, _ := filepath.Glob("/vagrant/share/*-crt.pem")
	for _, p := range paths {
		value, err := getCertificateFileText(p)
		if err != nil {
			value = fmt.Sprintf("ERROR: %v", err)
		}
		data = append(data, []string{path.Base(p), string(value)})
	}

	return &table{
		header: []string{"Certificate", "Content"},
		data:   data,
	}, nil
}

func main() {
	t, err := getTPMInfo()
	if err != nil {
		log.Fatalf("failed to get TPM Info: %v", err)
	}
	t.Render()

	t, err = getTPMEndorsementKeys()
	if err != nil {
		log.Fatalf("failed to get TPM Endorsement Keys: %v", err)
	}
	t.Render()

	t, err = getSwtpmCertificates()
	if err != nil {
		log.Fatalf("failed to get SWTPM Certificates: %v", err)
	}
	t.Render()
}
