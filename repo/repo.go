package repo

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jinzhu/copier"

	"fileup/constants"
)

type Repository interface {
	// Init creates a file for appending in wronly mode
	Init(string) error
	// Close performs cleanup upon file close
	Close(filename string)
	// SaveChunk appends data to file
	SaveChunk(filename string, chunk []byte) error
	// ProcessJSON processes JSON input file:
	// properties starting with vowels are removed.
	// properties with even ints incremented by 1000.
	// processed results persisted in a new json file on disk.
	// returned values are new filename and error
	ProcessJSON(srcFile string) (string, error)
}

type repo struct {
	fileDataMu sync.RWMutex
	fileData   map[string]fileRecord
	log        *log.Logger
	// path is the path to folder where files are uploaded
	path string
	// path is the path to folder where processed files are stored
	processedJSONPath string
}

type fileRecord struct {
	handle *os.File
	// createdAt can be used by a cleanup function to cleanup old entries
	createdAt time.Time
}

type entrySchema struct {
	PlayerName  string `json:"playerName,omitempty"`
	AvatarName  string `json:"avatarName,omitempty"`
	PlayerScore int    `json:"playerScore,omitempty"`
	LifeCount   int    `json:"lifeCount,omitempty"`
	Game        string `json:"game,omitempty"`
}

func NewRepo(log *log.Logger, path, processedJSONPath string) Repository {
	return &repo{
		fileData:          make(map[string]fileRecord),
		log:               log,
		path:              path,
		processedJSONPath: processedJSONPath,
	}
}

func (r *repo) Init(filename string) error {
	r.fileDataMu.Lock()
	defer r.fileDataMu.Unlock()

	filePath := fmt.Sprintf("%s/%s", r.path, filename)

	// check if file currently open
	// if yes, do nothing
	_, ok := r.fileData[filename]
	if ok {
		return nil
	}

	// 0644 = rw-r--r--
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("error during file:%s create:%v", filePath, err)
	}

	r.fileData[filename] = fileRecord{
		handle:    f,
		createdAt: time.Now(),
	}

	r.log.Printf("file %s created", filePath)

	return nil
}

func (r *repo) Close(filename string) {
	r.fileDataMu.Lock()
	defer r.fileDataMu.Unlock()

	// close file and delete entry
	handle := r.fileData[filename].handle
	defer handle.Close()
	delete(r.fileData, filename)

	// copy to avoid map leak
	dup := make(map[string]fileRecord)
	if err := copier.Copy(dup, r.fileData); err == nil {
		r.fileData = dup
	}
}

func (r *repo) SaveChunk(filename string, chunk []byte) error {
	r.fileDataMu.RLock()
	defer r.fileDataMu.RUnlock()

	record, ok := r.fileData[filename]
	if !ok {
		return fmt.Errorf("file not found. Save unsuccessful")
	}

	_, err := record.handle.Write(chunk)
	if err != nil {
		return fmt.Errorf("could not save chunk: %v", err)
	}

	return nil
}

func (r *repo) ProcessJSON(srcFile string) (string, error) {
	// open source JSON file to process
	srcFilePath := fmt.Sprintf("%s/%s", r.path, srcFile)
	f, err := os.Open(srcFilePath)
	if err != nil {
		return "", fmt.Errorf("error opening JSON file %s for processing:%v", srcFile, err)
	}
	defer close(f, fmt.Sprintf("error when closing JSON source file:%s", srcFile))

	id, err := uuid.NewRandom()
	if err != nil {
		return "", fmt.Errorf("could not generate JSON filename:%v", err)
	}

	filename := fmt.Sprintf("%s.json", id.String())
	filePath := fmt.Sprintf("%s/%s", r.processedJSONPath, filename)

	// 0644 = rw-r--r--
	wf, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return "", fmt.Errorf("error during file:%s create:%v", filePath, err)
	}
	defer close(wf, fmt.Sprintf("error when closing processed JSON file:%s", filename))

	// decode tokens
	rd := bufio.NewReader(f)
	dec := json.NewDecoder(rd)

	newJson := []entrySchema{}

	dec.Token()
	for dec.More() {
		var entry entrySchema

		if err := dec.Decode(&entry); err != nil {
			return "", fmt.Errorf("error occured when decoding JSON:%v", err)
		}

		newEntry := entrySchema{}
		if !skipEntry(entry.PlayerName) {
			newEntry.PlayerName = entry.PlayerName
		}

		if !skipEntry(entry.AvatarName) {
			newEntry.AvatarName = entry.AvatarName
		}

		if !skipEntry(entry.Game) {
			newEntry.Game = entry.Game
		}

		// note: zero is considered even number
		if shouldIncrement(entry.PlayerScore) {
			newEntry.PlayerScore = increment(entry.PlayerScore)
		} else {
			newEntry.PlayerScore = entry.PlayerScore
		}

		// note: zero is considered even number
		if shouldIncrement(entry.LifeCount) {
			newEntry.LifeCount = increment(entry.LifeCount)
		} else {
			newEntry.LifeCount = entry.LifeCount
		}

		newJson = append(newJson, newEntry)

	}

	newJsonEnc, err := json.Marshal(newJson)
	if err != nil {
		return "", fmt.Errorf("error occured in marshaling entry:%+v err:%v", newJson, err)
	}

	_, err = wf.Write(newJsonEnc)
	if err != nil {
		return "", fmt.Errorf("error occured in writing to json file:%v", err)
	}

	r.log.Printf("file %s processed to %s", srcFile, filename)

	return filename, nil
}

func close(f *os.File, errmsg string) {
	if err := f.Close(); err != nil {
		log.Printf("%s err:%s", errmsg, err)
	}
}

func skipEntry(entry string) bool {
	// false if entry empty
	if entry == "" {
		return false
	}

	skipList := []string{"a", "e", "i", "o", "u"}
	firstChar := strings.ToLower(entry[0:1])
	skip := false

	for _, val := range skipList {
		if firstChar == val {
			skip = true
			break
		}
	}

	return skip
}

func shouldIncrement(val int) bool {
	return val%2 == 0
}

func increment(val int) int {
	return val + constants.Increment
}
