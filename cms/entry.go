package cms

import (
	"path/filepath"
	"strings"
)

type ContentType string

const (
	ContentTypeMarkdown  ContentType = "text/markdown"
	ContentTypeJSON      ContentType = "application/json"
	ContentTypeText      ContentType = "text/plain"
	ContentTypeImagePNG  ContentType = "image/png"
	ContentTypeImageJPEG ContentType = "image/jpeg"
	ContentTypeUnknown   ContentType = "unknown"
)

func ContentTypeFromPath(path string) ContentType {
	ext := filepath.Ext(path)
	switch strings.ToLower(ext) {
	case ".md":
		return ContentTypeMarkdown
	case ".json":
		return ContentTypeJSON
	case ".txt":
		return ContentTypeText
	case ".png":
		return ContentTypeImagePNG
	case ".jpeg", ".jpg":
		return ContentTypeImageJPEG
	default:
		return ContentTypeUnknown
	}
}

type Entry struct {
	Name        string
	ContentType ContentType
	Payload     []byte
}
