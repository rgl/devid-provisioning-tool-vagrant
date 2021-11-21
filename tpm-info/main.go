package main

import (
	"bufio"
	"bytes"
	"crypto/x509"
	"encoding/asn1"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

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

func splitLines(s string) []string {
	var lines []string
	sc := bufio.NewScanner(strings.NewReader(s))
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	return lines
}

func trimLines(text string, maxWith int) string {
	lines := splitLines(text)
	for i, v := range lines {
		if len(v) > maxWith {
			lines[i] = v[0:maxWith-5] + " (...)"
		}
	}
	return strings.Join(lines, "\n")
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

func loadCertificate(path string) (*x509.Certificate, error) {
	raw, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode([]byte(raw))
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM from %s", path)
	}
	certificate, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate from %s: %w", path, err)
	}
	return certificate, nil
}

func getDevIDCertificate() (*table, error) {
	data := make([][]string, 0)

	for _, p := range []string{"conf/server/provisioning-ca.crt", "out/devid-certificate.pem"} {
		value, err := getCertificateFileText(p)
		if err != nil {
			value = fmt.Sprintf("ERROR: %v", err)
		}
		data = append(data, []string{path.Base(p), trimLines(string(value), 72)})
	}

	provisioningCACertificate, err := loadCertificate("conf/server/provisioning-ca.crt")
	if err != nil {
		return nil, err
	}
	devIDCertificate, err := loadCertificate("out/devid-certificate.pem")
	if err != nil {
		return nil, err
	}

	// verify whether the DevID certificate chain is valid (e.g. was signed by the provisioning CA).
	devIDExtKeyUsageOID := asn1.ObjectIdentifier{2, 23, 133, 11, 1, 2}
	hasDevIDExtKeyUsageOID := false
	for _, v := range devIDCertificate.UnknownExtKeyUsage {
		if v.Equal(devIDExtKeyUsageOID) {
			hasDevIDExtKeyUsageOID = true
		}
	}
	if !hasDevIDExtKeyUsageOID {
		data = append(data, []string{"DevID Certificate Verify", fmt.Sprintf("Failed with: Does not have the DevID ExtKeyUsage OID %s", devIDExtKeyUsageOID)})
	}
	roots := x509.NewCertPool()
	roots.AddCert(provisioningCACertificate)
	chains, err := devIDCertificate.Verify(x509.VerifyOptions{
		Roots:     roots,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
	})
	if err != nil {
		data = append(data, []string{"DevID Certificate Verify", fmt.Sprintf("Failed with: %v", err)})
	} else {
		data = append(data, []string{"DevID Certificate Verify", "Succeeded"})
		for i, chain := range chains {
			for j, c := range chain {
				data = append(data, []string{fmt.Sprintf("DevID Certificate Chain #%d.%d", i, j), fmt.Sprintf("Subject: %s", c.Subject)})
			}
		}
	}

	return &table{
		header: []string{"Certificate", "Content"},
		data:   data,
	}, nil
}

func main() {
	t, err := getTPMInfo()
	if err != nil {
		log.Printf("WARN: Failed to get TPM Info: %v", err)
	} else {
		t.Render()
	}

	t, err = getTPMEndorsementKeys()
	if err != nil {
		log.Printf("WARN: Failed to get TPM Endorsement Keys: %v", err)
	} else {
		t.Render()
	}

	t, err = getSwtpmCertificates()
	if err != nil {
		log.Printf("WARN: Failed to get SWTPM Certificates: %v", err)
	} else {
		t.Render()
	}

	t, err = getDevIDCertificate()
	if err != nil {
		log.Printf("WARN: Failed to get DevID Certificate: %v", err)
	} else {
		t.Render()
	}
}
