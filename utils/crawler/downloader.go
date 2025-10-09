package crawler

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func DownloadToBytes(rawURL string) ([]byte, error) {
	client := &http.Client{
		Timeout: 15 * time.Second,
	}

	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "WgetGo/1.0 (+https://example.com)")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status: %s", resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body error: %w", err)
	}
	return data, nil
}

func SaveFile(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return fmt.Errorf("mkdir error: %w", err)
	}
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create file error: %w", err)
	}
	defer f.Close()
	if _, err := f.Write(data); err != nil {
		return fmt.Errorf("write file error: %w", err)
	}
	return nil
}

func uniqueNameFromURL(rawURL, ext string) string {
	h := sha1.Sum([]byte(rawURL))
	name := hex.EncodeToString(h[:])
	if ext != "" && !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	return name + ext
}
