package pcsweb

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

const sessionCookieName = "pcsweb_session"

type webUploadTask struct {
	ID        string
	SessionID string
	Path      string
	Length    int64
	Status    string
	StartedAt int64
}

type webDownloadTask struct {
	ID        string
	SessionID string
	Path      string
	SavePath  string
	Status    string
	StartedAt int64
}

var (
	uploadExecutionMu sync.Mutex
	uploadTasksMu     sync.Mutex
	uploadTasks       = make(map[string]webUploadTask)
	downloadTasksMu   sync.Mutex
	downloadTasks     = make(map[string]webDownloadTask)
	taskSequence      uint64
)

func ensureSession(w http.ResponseWriter, r *http.Request) string {
	if cookie, err := r.Cookie(sessionCookieName); err == nil && cookie.Value != "" {
		return cookie.Value
	}
	data := make([]byte, 16)
	if _, err := rand.Read(data); err != nil {
		data = []byte(time.Now().Format("20060102150405.000000000"))
	}
	sessionID := hex.EncodeToString(data)
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    sessionID,
		Path:     "/",
		MaxAge:   30 * 24 * 60 * 60,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	return sessionID
}

func addUploadTask(sessionID, path string, length int64) string {
	task := webUploadTask{ID: newTaskID(), SessionID: sessionID, Path: path, Length: length, Status: "排队中", StartedAt: time.Now().Unix()}
	uploadTasksMu.Lock()
	uploadTasks[task.ID] = task
	uploadTasksMu.Unlock()
	return task.ID
}

func updateUploadTask(taskID, status string) {
	uploadTasksMu.Lock()
	defer uploadTasksMu.Unlock()
	task, ok := uploadTasks[taskID]
	if ok {
		task.Status = status
		uploadTasks[taskID] = task
	}
}

func removeUploadTasks(taskIDs []string) {
	uploadTasksMu.Lock()
	defer uploadTasksMu.Unlock()
	for _, taskID := range taskIDs {
		delete(uploadTasks, taskID)
	}
}

func listUploadTasks(sessionID string) []webUploadTask {
	uploadTasksMu.Lock()
	defer uploadTasksMu.Unlock()
	tasks := make([]webUploadTask, 0)
	for _, task := range uploadTasks {
		if task.SessionID == sessionID {
			tasks = append(tasks, task)
		}
	}
	return tasks
}

func addDownloadTask(sessionID, path, savePath string) string {
	task := webDownloadTask{ID: newTaskID(), SessionID: sessionID, Path: path, SavePath: savePath, Status: "下载中", StartedAt: time.Now().Unix()}
	downloadTasksMu.Lock()
	downloadTasks[task.ID] = task
	downloadTasksMu.Unlock()
	return task.ID
}

func removeDownloadTask(taskID string) {
	downloadTasksMu.Lock()
	delete(downloadTasks, taskID)
	downloadTasksMu.Unlock()
}

func listDownloadTasks(sessionID string) []webDownloadTask {
	downloadTasksMu.Lock()
	defer downloadTasksMu.Unlock()
	tasks := make([]webDownloadTask, 0)
	for _, task := range downloadTasks {
		if task.SessionID == sessionID {
			tasks = append(tasks, task)
		}
	}
	return tasks
}

func downloadPathActive(savePath string) bool {
	downloadTasksMu.Lock()
	defer downloadTasksMu.Unlock()
	for _, task := range downloadTasks {
		if task.SavePath == savePath {
			return true
		}
	}
	return false
}

func newTaskID() string {
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), atomic.AddUint64(&taskSequence, 1))
}
