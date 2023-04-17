package repo

import (
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProcessJSON(t *testing.T) {
	logger := log.New(os.Stdout, "", log.Lmsgprefix)
	fileDir := "../tmp/client"
	processedJSONfileDir := "../tmp/json"

	repo := NewRepo(logger, fileDir, processedJSONfileDir)

	filename, err := repo.ProcessJSON("testing.json")
	require.NoError(t, err)

	savedFilePath := fmt.Sprintf("%s/%s", processedJSONfileDir, filename)
	require.FileExists(t, savedFilePath)

	require.NoError(t, os.Remove(savedFilePath))
}
