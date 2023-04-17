package service

import (
	"context"
	"fileup/constants"
	"fileup/repo"
	"fmt"
	"log"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

type Service interface {
	// Init generates filename and calls repo Init
	Init(filetype string) (string, error)
	Close(filename string)
	SaveChunk(filename string, chunk []byte) error
}

type service struct {
	repo repo.Repository
	log  *log.Logger
}

func NewService(repo repo.Repository, log *log.Logger) Service {
	return &service{
		repo: repo,
		log:  log,
	}
}

func (svc *service) Init(filetype string) (string, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return "", fmt.Errorf("could not generate filename:%v", err)
	}

	filename := fmt.Sprintf("%s%s", id.String(), filetype)

	err = svc.repo.Init(filename)
	if err != nil {
		return "", err
	}

	return filename, nil
}

func (svc *service) SaveChunk(filename string, chunk []byte) error {
	return svc.repo.SaveChunk(filename, chunk)
}

func (svc *service) Close(filename string) {
	svc.repo.Close(filename)
	// in a production system, its ideal to do this post-processing in a background worker service.
	// for the scope of this test, we are using fire-forget pattern.
	// assumption is made that post-processing results are not part of response
	if filepath.Ext(filename) == constants.JSONExt {
		go svc.processJSON(filename)
	}
}

// processJSON logs the fact goroutine processing JSON is taking too long
func (svc *service) processJSON(filename string) {
	errCh := make(chan error)

	ctx, ctxCancel := context.WithTimeout(context.Background(), time.Second)
	defer ctxCancel()

	go func() {
		_, err := svc.repo.ProcessJSON(filename)
		errCh <- err
	}()

	select {
	case <-ctx.Done():
		log.Printf("Could npt process JSON for file:%s due to Timeout", filename)
	case err := <-errCh:
		if err != nil {
			log.Printf("error in processing JSON for file:%s err:%v", filename, err)
		}
	}

}
