/*package main

import (
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
	config.LocalID = raft.ServerID("node1")

	addr, err := raft.NewTCPTransport("127.0.0.1:9001", nil, 2, 10*time.Second, os.Stderr)
	if err != nil {
		log.Fatalf("Failed to create transport: %v", err)
	}

	dbStore, err := raftboltdb.NewBoltStore("raft-log1.bolt")
	if err != nil {
		log.Fatalf("Failed to create log store: %v", err)
	}

	stableStore, err := raftboltdb.NewBoltStore("raft-stable1.bolt")
	if err != nil {
		log.Fatalf("Failed to create stable store: %v", err)
	}

	snapshotStore := raft.NewDiscardSnapshotStore()

	fsm = NewFSM()
	raftNode, err = raft.NewRaft(config, fsm, dbStore, stableStore, snapshotStore, addr)
	if err != nil {
		log.Fatalf("Failed to create Raft node: %v", err)
	}

	logrus.Info("Raft Node 1 Initialized")

	router := mux.NewRouter()
	router.HandleFunc("/status", getRaftStatus).Methods("GET")
	router.HandleFunc("/submit", submitJob).Methods("POST")
	router.HandleFunc("/join", joinCluster).Methods("POST")
	router.HandleFunc("/jobs", getJobs).Methods("GET")

	// Enable CORS for frontend (e.g., 127.0.0.1:5500)
	corsHandler := cors.Default().Handler(router)

	log.Fatal(http.ListenAndServe(":8080", corsHandler))
}

// Structs and Handlers

type JoinRequest struct {
	ID   string `json:"id"`
	Addr string `json:"addr"`
}

func joinCluster(w http.ResponseWriter, r *http.Request) {
	var req JoinRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid join request", http.StatusBadRequest)
		return
	}
	logrus.Infof("Received join request from %s at %s", req.ID, req.Addr)
	future := raftNode.AddVoter(raft.ServerID(req.ID), raft.ServerAddress(req.Addr), 0, 0)
	if err := future.Error(); err != nil {
		http.Error(w, fmt.Sprintf("Failed to add voter: %v", err), http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "Node %s joined successfully", req.ID)
}

func getRaftStatus(w http.ResponseWriter, r *http.Request) {
	state := raftNode.State()
	fmt.Fprintf(w, "Node State: %s", state.String())
}

func submitJob(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	command := string(body)
	logrus.Infof("Received job: %s", command)

	if raftNode.State() != raft.Leader {
		http.Error(w, "Not the leader", http.StatusForbidden)
		return
	}

	f := raftNode.Apply([]byte(command), 5*time.Second)
	if err := f.Error(); err != nil {
		http.Error(w, fmt.Sprintf("Failed to apply: %v", err), http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "Command submitted: %s\n", command)
}

func getJobs(w http.ResponseWriter, r *http.Request) {
	fsm.mu.Lock()
	defer fsm.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(fsm.jobs)
}

// FSM Implementation

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

func (s *snapshot) Release() {}*/

// main.go - Node 1 (Leader)
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
	"github.com/rs/cors"
)

var raftNode *raft.Raft
var fsm *FSM

func main() {
	// === RAFT SETUP ===
	config := raft.DefaultConfig()
	config.LocalID = raft.ServerID("node1")

	addr, err := raft.NewTCPTransport("127.0.0.1:9001", nil, 3, 10*time.Second, os.Stderr)
	if err != nil {
		log.Fatalf("Transport error: %v", err)
	}

	logStore, err := raftboltdb.NewBoltStore("raft-log1.bolt")
	if err != nil {
		log.Fatalf("Log store error: %v", err)
	}

	stableStore, err := raftboltdb.NewBoltStore("raft-stable1.bolt")
	if err != nil {
		log.Fatalf("Stable store error: %v", err)
	}

	snapshotStore := raft.NewDiscardSnapshotStore()

	fsm = NewFSM()
	raftNode, err = raft.NewRaft(config, fsm, logStore, stableStore, snapshotStore, addr)
	if err != nil {
		log.Fatalf("Raft init error: %v", err)
	}

	log.Println("✅ Node1 Raft ready on 127.0.0.1:9001")

	// === HTTP API ===
	r := mux.NewRouter()

	r.HandleFunc("/status", statusHandler).Methods("GET")
	r.HandleFunc("/join", joinHandler).Methods("POST")

	r.HandleFunc("/api/v1/printers", createPrinter).Methods("POST")
	r.HandleFunc("/api/v1/printers", getPrinters).Methods("GET")

	r.HandleFunc("/api/v1/filaments", createFilament).Methods("POST")
	r.HandleFunc("/api/v1/filaments", getFilaments).Methods("GET")

	r.HandleFunc("/api/v1/print_jobs", createJob).Methods("POST")
	r.HandleFunc("/api/v1/print_jobs", getJobs).Methods("GET")

	r.HandleFunc("/api/v1/print_jobs/{id}/status", updateJobStatus).Methods("POST")

	h := cors.Default().Handler(r)
	log.Fatal(http.ListenAndServe(":8080", h))
}

// Join request from other nodes
type JoinRequest struct {
	ID   string `json:"id"`
	Addr string `json:"addr"`
}

func joinHandler(w http.ResponseWriter, r *http.Request) {
	if raftNode.State() != raft.Leader {
		http.Error(w, "Not the leader", 403)
		return
	}
	var req JoinRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid join payload", 400)
		return
	}
	f := raftNode.AddVoter(raft.ServerID(req.ID), raft.ServerAddress(req.Addr), 0, 0)
	if err := f.Error(); err != nil {
		http.Error(w, "Join failed: "+err.Error(), 500)
		return
	}
	fmt.Fprintf(w, "Node %s joined successfully", req.ID)
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Node State: %s", raftNode.State().String())
}

// === API HANDLERS ===

func createPrinter(w http.ResponseWriter, r *http.Request) {
	var p Printer
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	if err := applyCommand("create_printer", p, raftNode); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Write([]byte("Printer created"))
}

func getPrinters(w http.ResponseWriter, r *http.Request) {
	fsm.mu.Lock()
	defer fsm.mu.Unlock()
	json.NewEncoder(w).Encode(fsm.printers)
}

func createFilament(w http.ResponseWriter, r *http.Request) {
	var f Filament
	if err := json.NewDecoder(r.Body).Decode(&f); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	if err := applyCommand("create_filament", f, raftNode); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Write([]byte("Filament created"))
}

func getFilaments(w http.ResponseWriter, r *http.Request) {
	fsm.mu.Lock()
	defer fsm.mu.Unlock()
	json.NewEncoder(w).Encode(fsm.filaments)
}

func createJob(w http.ResponseWriter, r *http.Request) {
	var j PrintJob
	if err := json.NewDecoder(r.Body).Decode(&j); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	if err := applyCommand("create_job", j, raftNode); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Write([]byte("Print job created"))
}

func getJobs(w http.ResponseWriter, r *http.Request) {
	fsm.mu.Lock()
	defer fsm.mu.Unlock()
	json.NewEncoder(w).Encode(fsm.jobs)
}

func updateJobStatus(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	status := r.URL.Query().Get("status")
	payload := map[string]string{"job_id": id, "status": status}
	if err := applyCommand("update_job_status", payload, raftNode); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Write([]byte("Status updated"))
}
