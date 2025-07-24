/*package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
	"github.com/rs/cors"
	"github.com/sirupsen/logrus"
)

var raftNode *raft.Raft
var fsm *FSM

func main() {
	config := raft.DefaultConfig()
	config.LocalID = raft.ServerID("node3")

	addr, err := raft.NewTCPTransport("127.0.0.1:9003", nil, 2, 10*time.Second, os.Stderr)
	if err != nil {
		log.Fatalf("Failed to create transport: %v", err)
	}

	dbStore, err := raftboltdb.NewBoltStore("raft-log3.bolt")
	if err != nil {
		log.Fatalf("Failed to create log store: %v", err)
	}

	stableStore, err := raftboltdb.NewBoltStore("raft-stable3.bolt")
	if err != nil {
		log.Fatalf("Failed to create stable store: %v", err)
	}

	snapshotStore := raft.NewDiscardSnapshotStore()

	fsm = NewFSM()
	raftNode, err = raft.NewRaft(config, fsm, dbStore, stableStore, snapshotStore, addr)
	if err != nil {
		log.Fatalf("Failed to create Raft node: %v", err)
	}

	logrus.Info("Raft Node 3 Initialized")

	joinLeader()

	router := mux.NewRouter()
	router.HandleFunc("/status", getRaftStatus).Methods("GET")
	router.HandleFunc("/jobs", getJobs).Methods("GET")

	corsHandler := cors.Default().Handler(router)
	log.Fatal(http.ListenAndServe(":8082", corsHandler))
}

func joinLeader() {
	payload := map[string]string{
		"id":   "node3",
		"addr": "127.0.0.1:9003",
	}
	body, _ := json.Marshal(payload)
	resp, err := http.Post("http://127.0.0.1:8080/join", "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("Error joining cluster: %v", err)
		return
	}
	defer resp.Body.Close()
	content, _ := io.ReadAll(resp.Body)
	log.Printf("Join response: %s", string(content))
}

func getRaftStatus(w http.ResponseWriter, r *http.Request) {
	state := raftNode.State()
	fmt.Fprintf(w, "Node State: %s", state.String())
}

func getJobs(w http.ResponseWriter, r *http.Request) {
	fsm.mu.Lock()
	defer fsm.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(fsm.jobs)
}

type FSM struct {
	jobs []string
	mu   sync.Mutex
}

func NewFSM() *FSM {
	f := &FSM{jobs: []string{}}
	content, err := os.ReadFile("job_log.txt")
	if err == nil {
		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			if line != "" {
				f.jobs = append(f.jobs, line)
			}
		}
	}
	return f
}

func (f *FSM) Apply(log *raft.Log) interface{} {
	f.mu.Lock()
	defer f.mu.Unlock()
	job := string(log.Data)
	f.jobs = append(f.jobs, job)
	f.persistJob(job)
	fmt.Printf("FSM Apply: %s\n", job)
	return nil
}

func (f *FSM) persistJob(job string) {
	file, err := os.OpenFile("job_log.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err == nil {
		defer file.Close()
		file.WriteString(job + "\n")
	}
}

func (f *FSM) Snapshot() (raft.FSMSnapshot, error) {
	return &snapshot{}, nil
}

func (f *FSM) Restore(rc io.ReadCloser) error {
	return nil
}

type snapshot struct{}

func (s *snapshot) Persist(sink raft.SnapshotSink) error {
	sink.Cancel()
	return nil
}

func (s *snapshot) Release() {}


// node3.go

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
	"github.com/rs/cors"
	"github.com/sirupsen/logrus"
)

var raftNode *raft.Raft
var fsm *FSM

func main() {
	config := raft.DefaultConfig()
	config.LocalID = raft.ServerID("node3")

	// Transport for node3
	addr, err := raft.NewTCPTransport("127.0.0.1:9003", nil, 2, 10*time.Second, os.Stderr)
	if err != nil {
		log.Fatalf("Failed to create transport: %v", err)
	}

	// Raft log store
	logStore, err := raftboltdb.NewBoltStore("raft-log3.bolt")
	if err != nil {
		log.Fatalf("Failed to create log store: %v", err)
	}

	// Raft stable store
	stableStore, err := raftboltdb.NewBoltStore("raft-stable3.bolt")
	if err != nil {
		log.Fatalf("Failed to create stable store: %v", err)
	}

	// Snapshot store
	snapshotStore, err := raft.NewFileSnapshotStore(".", 1, os.Stderr)
	if err != nil {
		log.Fatalf("Failed to create snapshot store: %v", err)
	}

	// FSM
	fsm = NewFSM()

	// Raft node
	raftNode, err = raft.NewRaft(config, fsm, logStore, stableStore, snapshotStore, addr)
	if err != nil {
		log.Fatalf("Failed to create Raft node: %v", err)
	}

	logrus.Info("Raft Node 3 Initialized")

	// Join the leader (node1)
	joinLeader()

	// HTTP API for Raft status and jobs
	router := mux.NewRouter()
	router.HandleFunc("/status", getRaftStatus).Methods("GET")
	router.HandleFunc("/jobs", getJobs).Methods("GET")

	// Enable CORS
	corsHandler := cors.Default().Handler(router)
	log.Fatal(http.ListenAndServe(":8082", corsHandler))
}

func joinLeader() {
	payload := map[string]string{
		"id":   "node3",
		"addr": "127.0.0.1:9003",
	}
	body, _ := json.Marshal(payload)
	resp, err := http.Post("http://127.0.0.1:8080/join", "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("Error joining cluster: %v", err)
		return
	}
	defer resp.Body.Close()
	content, _ := io.ReadAll(resp.Body)
	log.Printf("Join response: %s", string(content))
}

func getRaftStatus(w http.ResponseWriter, r *http.Request) {
	state := raftNode.State()
	fmt.Fprintf(w, "Node State: %s", state.String())
}

func getJobs(w http.ResponseWriter, r *http.Request) {
	fsm.mu.Lock()
	defer fsm.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(fsm.jobs)
}*/

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
	"github.com/rs/cors"
	"github.com/sirupsen/logrus"
)

var raftNode *raft.Raft
var fsm *FSM

func main() {
	config := raft.DefaultConfig()
	config.LocalID = raft.ServerID("node3")

	addr, err := raft.NewTCPTransport("127.0.0.1:9003", nil, 2, 10*time.Second, os.Stderr)
	if err != nil {
		log.Fatalf("Failed to create transport: %v", err)
	}

	logStore, err := raftboltdb.NewBoltStore("raft-log3.bolt")
	if err != nil {
		log.Fatalf("Failed to create log store: %v", err)
	}

	stableStore, err := raftboltdb.NewBoltStore("raft-stable3.bolt")
	if err != nil {
		log.Fatalf("Failed to create stable store: %v", err)
	}

	snapshotStore, err := raft.NewFileSnapshotStore(".", 1, os.Stderr)
	if err != nil {
		log.Fatalf("Failed to create snapshot store: %v", err)
	}

	fsm = NewFSM()

	raftNode, err = raft.NewRaft(config, fsm, logStore, stableStore, snapshotStore, addr)
	if err != nil {
		log.Fatalf("Failed to create Raft node: %v", err)
	}

	logrus.Info("Raft Node 3 Initialized")

	joinLeader()

	router := mux.NewRouter()
	router.HandleFunc("/status", getRaftStatus).Methods("GET")
	router.HandleFunc("/api/v1/printers", getPrinters).Methods("GET")
	router.HandleFunc("/api/v1/filaments", getFilaments).Methods("GET")
	router.HandleFunc("/api/v1/print_jobs", getJobs).Methods("GET")

	corsHandler := cors.Default().Handler(router)
	log.Fatal(http.ListenAndServe(":8082", corsHandler))
}

func joinLeader() {
	payload := map[string]string{
		"id":   "node3",
		"addr": "127.0.0.1:9003",
	}
	body, _ := json.Marshal(payload)
	resp, err := http.Post("http://127.0.0.1:8080/join", "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("Error joining cluster: %v", err)
		return
	}
	defer resp.Body.Close()
	content, _ := io.ReadAll(resp.Body)
	log.Printf("Join response: %s", string(content))
}

func getRaftStatus(w http.ResponseWriter, r *http.Request) {
	state := raftNode.State()
	fmt.Fprintf(w, "Node State: %s", state.String())
}

func getPrinters(w http.ResponseWriter, r *http.Request) {
	fsm.mu.Lock()
	defer fsm.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(fsm.printers)
}

func getFilaments(w http.ResponseWriter, r *http.Request) {
	fsm.mu.Lock()
	defer fsm.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(fsm.filaments)
}

func getJobs(w http.ResponseWriter, r *http.Request) {
	fsm.mu.Lock()
	defer fsm.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(fsm.jobs)
}
