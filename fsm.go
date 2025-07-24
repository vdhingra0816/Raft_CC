/*package main

import (
	"encoding/json"
	"io"
	"log"
	"sync"

	"github.com/hashicorp/raft"
)

type PrintJob struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

type FSM struct {
	mu   sync.Mutex
	jobs map[string]PrintJob
}

func NewFSM() *FSM {
	return &FSM{
		jobs: make(map[string]PrintJob),
	}
}

func (f *FSM) Apply(logEntry *raft.Log) interface{} {
	var job PrintJob
	err := json.Unmarshal(logEntry.Data, &job)
	if err != nil {
		log.Printf("Failed to unmarshal job: %v", err)
		return nil
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	job.Status = "queued"
	f.jobs[job.ID] = job

	log.Printf("Job %s added to queue", job.ID)
	return nil
}

func (f *FSM) Snapshot() (raft.FSMSnapshot, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	jobsCopy := make(map[string]PrintJob)
	for k, v := range f.jobs {
		jobsCopy[k] = v
	}

	return &fsmSnapshot{jobs: jobsCopy}, nil
}

func (f *FSM) Restore(rc io.ReadCloser) error {
	var jobs map[string]PrintJob
	if err := json.NewDecoder(rc).Decode(&jobs); err != nil {
		return err
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	f.jobs = jobs
	return nil
}

type fsmSnapshot struct {
	jobs map[string]PrintJob
}

func (s *fsmSnapshot) Persist(sink raft.SnapshotSink) error {
	data, err := json.Marshal(s.jobs)
	if err != nil {
		sink.Cancel()
		return err
	}

	if _, err := sink.Write(data); err != nil {
		sink.Cancel()
		return err
	}
	return sink.Close()
}

func (s *fsmSnapshot) Release() {}*/

package main

import (
	"encoding/json"
	"io"
	"log"
	"sync"
	"time"

	"github.com/hashicorp/raft"
)

// =============================
// 📦 Structs for all resources
// =============================
type Printer struct {
	ID      string `json:"id"`
	Company string `json:"company"`
	Model   string `json:"model"`
}

type Filament struct {
	ID                     string `json:"id"`
	Type                   string `json:"type"`
	Color                  string `json:"color"`
	TotalWeightInGrams     int    `json:"total_weight_in_grams"`
	RemainingWeightInGrams int    `json:"remaining_weight_in_grams"`
}

type PrintJob struct {
	ID                 string `json:"id"`
	PrinterID          string `json:"printer_id"`
	FilamentID         string `json:"filament_id"`
	Filepath           string `json:"filepath"`
	PrintWeightInGrams int    `json:"print_weight_in_grams"`
	Status             string `json:"status"`
}

// =============================
// 📦 Command wrapper for FSM
// =============================
type RaftCommand struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

// =============================
// 🧠 FSM Implementation
// =============================
type FSM struct {
	mu        sync.Mutex
	printers  map[string]Printer
	filaments map[string]Filament
	jobs      map[string]PrintJob
}

func NewFSM() *FSM {
	return &FSM{
		printers:  make(map[string]Printer),
		filaments: make(map[string]Filament),
		jobs:      make(map[string]PrintJob),
	}
}

func (f *FSM) Apply(logEntry *raft.Log) interface{} {
	var cmd RaftCommand
	if err := json.Unmarshal(logEntry.Data, &cmd); err != nil {
		log.Printf("[FSM] Failed to unmarshal command: %v", err)
		return nil
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	switch cmd.Type {
	case "create_printer":
		var p Printer
		json.Unmarshal(cmd.Data, &p)
		f.printers[p.ID] = p

	case "create_filament":
		var fl Filament
		json.Unmarshal(cmd.Data, &fl)
		f.filaments[fl.ID] = fl

	case "create_job":
		var job PrintJob
		json.Unmarshal(cmd.Data, &job)

		if _, ok := f.printers[job.PrinterID]; !ok {
			log.Printf("[FSM] ❌ Printer not found: %s", job.PrinterID)
			return nil
		}

		fl, ok := f.filaments[job.FilamentID]
		if !ok {
			log.Printf("[FSM] ❌ Filament not found: %s (Job ID: %s)", job.FilamentID, job.ID)
			return nil
		}

		// ✅ Calculate RESERVED filament BEFORE adding this job
		reservedWeight := 0
		for _, j := range f.jobs {
			if j.FilamentID == job.FilamentID && (j.Status == "queued" || j.Status == "running") {
				reservedWeight += j.PrintWeightInGrams
			}
		}

		availableWeight := fl.RemainingWeightInGrams - reservedWeight
		if job.PrintWeightInGrams > availableWeight {
			log.Printf("[FSM] ❌ Not enough filament for Job %s: required %dg, reserved %dg, available %dg",
				job.ID, job.PrintWeightInGrams, reservedWeight, fl.RemainingWeightInGrams)
			return nil
		}

		// ✅ Now that check passed, reserve the filament
		job.Status = "queued"
		f.jobs[job.ID] = job
		log.Printf("[FSM] ✅ Job %s created and queued. Filament reserved: %d/%d grams",
			job.ID, reservedWeight+job.PrintWeightInGrams, fl.RemainingWeightInGrams)
	/*case "create_job":
	var job PrintJob
	json.Unmarshal(cmd.Data, &job)

	if _, ok := f.printers[job.PrinterID]; !ok {
		log.Printf("[FSM] Printer ID not found: %s", job.PrinterID)
		return nil
	}

	fl, ok := f.filaments[job.FilamentID]
	if !ok {
		log.Printf("[FSM] Filament ID not found: %s", job.FilamentID)
		return nil
	}

	// ✅ Calculate RESERVED filament BEFORE adding this job
	reservedWeight := 0
	for _, j := range f.jobs {
		if j.FilamentID == job.FilamentID && (j.Status == "queued" || j.Status == "running") {
			reservedWeight += j.PrintWeightInGrams
		}
	}

	availableWeight := fl.RemainingWeightInGrams - reservedWeight
	if job.PrintWeightInGrams > availableWeight {
		log.Printf("[FSM] ❌ Not enough filament: required %d, reserved %d, available %d", job.PrintWeightInGrams, reservedWeight, fl.RemainingWeightInGrams)
		return nil
	}

	// ✅ Now that check passed, reserve the filament
	job.Status = "queued"
	f.jobs[job.ID] = job
	log.Printf("[FSM] ✅ Job %s created and queued. Filament reserved: %d/%d", job.ID, reservedWeight+job.PrintWeightInGrams, fl.RemainingWeightInGrams)
	*/
	case "update_job_status":
		var payload struct {
			JobID  string `json:"job_id"`
			Status string `json:"status"`
		}
		json.Unmarshal(cmd.Data, &payload)
		f.updateJobStatus(payload.JobID, payload.Status)
	}
	return nil
}

func (f *FSM) updateJobStatus(jobID, status string) {
	job, exists := f.jobs[jobID]
	if !exists {
		log.Printf("[FSM] Job %s not found", jobID)
		return
	}

	valid := false
	switch status {
	case "running":
		valid = job.Status == "queued"
	case "done":
		valid = job.Status == "running"
	case "cancelled":
		valid = job.Status == "queued" || job.Status == "running"
	}

	if !valid {
		log.Printf("[FSM] Invalid transition: %s → %s", job.Status, status)
		return
	}

	job.Status = status

	if status == "done" {
		if fl, ok := f.filaments[job.FilamentID]; ok {
			if job.PrintWeightInGrams <= fl.RemainingWeightInGrams {
				fl.RemainingWeightInGrams -= job.PrintWeightInGrams
			} else {
				fl.RemainingWeightInGrams = 0
			}
			f.filaments[fl.ID] = fl
		}
	}

	f.jobs[jobID] = job
}

// =============================
// 📸 Snapshot & Restore
// =============================
type fsmSnapshot struct {
	Printers  map[string]Printer
	Filaments map[string]Filament
	Jobs      map[string]PrintJob
}

func (f *FSM) Snapshot() (raft.FSMSnapshot, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return &fsmSnapshot{
		Printers:  f.printers,
		Filaments: f.filaments,
		Jobs:      f.jobs,
	}, nil
}

func (f *FSM) Restore(rc io.ReadCloser) error {
	var snap fsmSnapshot
	if err := json.NewDecoder(rc).Decode(&snap); err != nil {
		return err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.printers = snap.Printers
	f.filaments = snap.Filaments
	f.jobs = snap.Jobs
	return nil
}

func (s *fsmSnapshot) Persist(sink raft.SnapshotSink) error {
	data, err := json.Marshal(s)
	if err != nil {
		sink.Cancel()
		return err
	}
	if _, err := sink.Write(data); err != nil {
		sink.Cancel()
		return err
	}
	return sink.Close()
}

func (s *fsmSnapshot) Release() {}

// =============================
// 🛠️ Helper for API calls
// =============================
func applyCommand(cmdType string, payload interface{}, r *raft.Raft) error {
	cmd := RaftCommand{
		Type: cmdType,
		Data: mustMarshal(payload),
	}
	data, _ := json.Marshal(cmd)
	f := r.Apply(data, 5*time.Second)
	return f.Error()
}

func mustMarshal(v interface{}) json.RawMessage {
	data, _ := json.Marshal(v)
	return data
}
