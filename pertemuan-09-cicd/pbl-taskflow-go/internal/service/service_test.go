package service_test

import (
	"testing"

	"github.com/taskflow/api/internal/model"
	"github.com/taskflow/api/internal/repository"
	"github.com/taskflow/api/internal/service"
)

func newSvc() *service.TaskService {
	return service.NewTaskService(repository.NewMemoryRepository())
}

// ── [BUG] CalculateCompletionRate ────────────────────────────────────────────
// BUG #1: Integer division — hasil selalu 0 (kecuali semua task selesai).

func TestCalculateCompletionRate(t *testing.T) {
	tests := []struct {
		name    string
		tasks   []model.Task
		want    float64
		isBug   bool
	}{
		{
			name:  "tidak ada task",
			tasks: []model.Task{},
			want:  0,
		},
		{
			name:  "semua done → 100%",
			tasks: []model.Task{{Status: model.StatusDone}, {Status: model.StatusDone}},
			want:  100.0,
		},
		{
			// [BUG] 1/3 dengan integer division = 0, bukan 33.33
			name: "[BUG] sepertiga selesai → 33.33%",
			tasks: []model.Task{
				{Status: model.StatusDone},
				{Status: model.StatusTodo},
				{Status: model.StatusTodo},
			},
			want:  33.33,
			isBug: true,
		},
		{
			// [BUG] 1/2 dengan integer division = 0, bukan 50.0
			name:  "[BUG] setengah selesai → 50%",
			tasks: []model.Task{{Status: model.StatusDone}, {Status: model.StatusTodo}},
			want:  50.0,
			isBug: true,
		},
		{
			name: "tidak ada yang selesai → 0%",
			tasks: []model.Task{
				{Status: model.StatusTodo},
				{Status: model.StatusInProgress},
			},
			want: 0.0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := service.CalculateCompletionRate(tc.tasks)
			// Toleransi 0.01 untuk floating point
			diff := got - tc.want
			if diff < 0 {
				diff = -diff
			}
			if diff > 0.01 {
				if tc.isBug {
					t.Errorf("BUG TERDETEKSI — CalculateCompletionRate() = %.2f, want %.2f\n"+
						"  → Integer division: %d/%d = 0 (bukan %.2f)\n"+
						"  → Perbaiki: gunakan float64(completed)/float64(len(tasks))*100",
						got, tc.want, len(tc.tasks)/2, len(tc.tasks), tc.want)
				} else {
					t.Errorf("CalculateCompletionRate() = %.2f, want %.2f", got, tc.want)
				}
			}
		})
	}
}

// ── Create ───────────────────────────────────────────────────────────────────

func TestCreate(t *testing.T) {
	svc := newSvc()

	t.Run("sukses dengan default priority", func(t *testing.T) {
		task, err := svc.Create(model.CreateTaskRequest{Title: "Belajar Go"})
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		if task.Title != "Belajar Go" {
			t.Errorf("Title = %q, want %q", task.Title, "Belajar Go")
		}
		if task.Status != model.StatusTodo {
			t.Errorf("Status = %q, want todo", task.Status)
		}
		if task.Priority != model.PriorityMedium {
			t.Errorf("Priority = %q, want medium (default)", task.Priority)
		}
		if task.ID == "" {
			t.Error("ID tidak boleh kosong")
		}
	})

	t.Run("title kosong ditolak", func(t *testing.T) {
		_, err := svc.Create(model.CreateTaskRequest{Title: ""})
		if err == nil {
			t.Error("Create() harus error jika title kosong")
		}
	})

	t.Run("title spasi saja ditolak", func(t *testing.T) {
		_, err := svc.Create(model.CreateTaskRequest{Title: "   "})
		if err == nil {
			t.Error("Create() harus error jika title hanya spasi")
		}
	})

	t.Run("priority invalid ditolak", func(t *testing.T) {
		_, err := svc.Create(model.CreateTaskRequest{Title: "T", Priority: "extreme"})
		if err == nil {
			t.Error("Create() harus error untuk priority tidak valid")
		}
	})

	t.Run("priority high sukses", func(t *testing.T) {
		task, err := svc.Create(model.CreateTaskRequest{Title: "Urgent", Priority: model.PriorityHigh})
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		if task.Priority != model.PriorityHigh {
			t.Errorf("Priority = %q, want high", task.Priority)
		}
	})

	t.Run("setiap task ID unik", func(t *testing.T) {
		ids := make(map[string]bool)
		for i := 0; i < 50; i++ {
			task, _ := svc.Create(model.CreateTaskRequest{Title: "Task"})
			if ids[task.ID] {
				t.Errorf("ID duplikat ditemukan: %s", task.ID)
			}
			ids[task.ID] = true
		}
	})
}

// ── Update ───────────────────────────────────────────────────────────────────

func TestUpdate(t *testing.T) {
	svc := newSvc()

	t.Run("update status ke done mengisi completed_at", func(t *testing.T) {
		task, _ := svc.Create(model.CreateTaskRequest{Title: "Selesaikan"})
		statusDone := model.StatusDone
		updated, err := svc.Update(task.ID, model.UpdateTaskRequest{Status: &statusDone})
		if err != nil {
			t.Fatalf("Update() error = %v", err)
		}
		if updated.CompletedAt == nil {
			t.Error("CompletedAt harus terisi setelah status = done")
		}
	})

	t.Run("update task tidak ada → error", func(t *testing.T) {
		statusDone := model.StatusDone
		_, err := svc.Update("id-tidak-ada", model.UpdateTaskRequest{Status: &statusDone})
		if err == nil {
			t.Error("Update() harus error untuk ID tidak ada")
		}
	})

	t.Run("update status invalid → error", func(t *testing.T) {
		task, _ := svc.Create(model.CreateTaskRequest{Title: "T"})
		s := model.Status("invalid")
		_, err := svc.Update(task.ID, model.UpdateTaskRequest{Status: &s})
		if err == nil {
			t.Error("Update() harus error untuk status tidak valid")
		}
	})
}

// ── [CICD] Full Task Lifecycle ────────────────────────────────────────────────
// [CICD] Simulasi integration test: create → get → update → delete.
// Jenis test ini dijalankan otomatis setelah deploy ke staging.

func TestTaskFullLifecycle(t *testing.T) {
	svc := newSvc()

	// 1. Create
	task, err := svc.Create(model.CreateTaskRequest{
		Title:    "Pipeline Lifecycle Test",
		Priority: model.PriorityHigh,
	})
	if err != nil {
		t.Fatalf("Create() gagal: %v", err)
	}

	// 2. Get
	got, err := svc.GetByID(task.ID)
	if err != nil || got.ID != task.ID {
		t.Fatalf("GetByID() gagal setelah create")
	}

	// 3. Update ke in_progress
	s := model.StatusInProgress
	got, err = svc.Update(task.ID, model.UpdateTaskRequest{Status: &s})
	if err != nil || got.Status != model.StatusInProgress {
		t.Fatalf("Update() ke in_progress gagal")
	}

	// 4. Update ke done
	done := model.StatusDone
	got, err = svc.Update(task.ID, model.UpdateTaskRequest{Status: &done})
	if err != nil || got.CompletedAt == nil {
		t.Fatalf("Update() ke done gagal atau CompletedAt nil")
	}

	// 5. Stats harus menunjukkan 1 done
	stats, err := svc.GetStats()
	if err != nil {
		t.Fatalf("GetStats() gagal: %v", err)
	}
	if stats.ByStatus["done"] != 1 {
		t.Errorf("Stats.ByStatus[done] = %d, want 1", stats.ByStatus["done"])
	}

	// 6. Delete
	_, err = svc.Delete(task.ID)
	if err != nil {
		t.Fatalf("Delete() gagal: %v", err)
	}

	// 7. Pastikan sudah terhapus
	if _, err = svc.GetByID(task.ID); err == nil {
		t.Error("GetByID() harus error setelah task dihapus")
	}
}

// ── [CICD] Rollback Simulation ───────────────────────────────────────────────

func TestRollbackStatusSimulation(t *testing.T) {
	svc := newSvc()
	task, _ := svc.Create(model.CreateTaskRequest{Title: "Rollback Test"})

	// Simulasi: deploy berhasil → update ke in_progress
	s := model.StatusInProgress
	svc.Update(task.ID, model.UpdateTaskRequest{Status: &s}) //nolint

	// Deployment bermasalah → rollback ke todo
	todo := model.StatusTodo
	rolled, err := svc.Update(task.ID, model.UpdateTaskRequest{Status: &todo})
	if err != nil {
		t.Fatalf("Rollback gagal: %v", err)
	}
	if rolled.Status != model.StatusTodo {
		t.Errorf("Setelah rollback, status = %q, want todo", rolled.Status)
	}
}

// ── TestCreate_WithUnicodeTitle ─────────────────────────────────────────────

func TestCreate_WithUnicodeTitle(t *testing.T) {
	svc := newSvc()

	unicodeTitle := "Learning Go 🚀 日本語 العربية"

	task, err := svc.Create(model.CreateTaskRequest{
		Title: unicodeTitle,
	})

	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Verify title is stored correctly
	if task.Title != unicodeTitle {
		t.Errorf("Title = %q, want %q", task.Title, unicodeTitle)
	}

	// Verify generated ID is not empty
	if task.ID == "" {
		t.Error("Task ID should not be empty")
	}
}

// ── TestDelete_AndVerifyStats ──────────────────────────────────────────────

func TestDelete_AndVerifyStats(t *testing.T) {
	svc := newSvc()

	// Create first task
	task1, err := svc.Create(model.CreateTaskRequest{
		Title: "Task 1",
	})
	if err != nil {
		t.Fatalf("Create() task1 error = %v", err)
	}

	// Create second task
	_, err = svc.Create(model.CreateTaskRequest{
		Title: "Task 2",
	})
	if err != nil {
		t.Fatalf("Create() task2 error = %v", err)
	}

	// Get stats before delete
	statsBefore, err := svc.GetStats()
	if err != nil {
		t.Fatalf("GetStats() before delete error = %v", err)
	}

	// Delete one task
	_, err = svc.Delete(task1.ID)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Get stats after delete
	statsAfter, err := svc.GetStats()
	if err != nil {
		t.Fatalf("GetStats() after delete error = %v", err)
	}

	// Verify total tasks decreased by 1
	if statsAfter.Total != statsBefore.Total-1 {
		t.Errorf(
			"Total stats after delete = %d, want %d",
			statsAfter.Total,
			statsBefore.Total-1,
		)
	}
}

// ── TestGetAll_WithStatusFilter ─────────────────────────────────────────────

func TestGetAll_WithStatusFilter(t *testing.T) {
	svc := newSvc()

	// Buat task dengan berbagai status
	task1, _ := svc.Create(model.CreateTaskRequest{Title: "Task Todo 1"})
	task2, _ := svc.Create(model.CreateTaskRequest{Title: "Task Todo 2"})
	task3, _ := svc.Create(model.CreateTaskRequest{Title: "Task In Progress"})
	task4, _ := svc.Create(model.CreateTaskRequest{Title: "Task Done"})

	// Update status task3 ke in_progress
	sIP := model.StatusInProgress
	svc.Update(task3.ID, model.UpdateTaskRequest{Status: &sIP}) //nolint

	// Update status task4 ke done
	sDone := model.StatusDone
	svc.Update(task4.ID, model.UpdateTaskRequest{Status: &sDone}) //nolint

	// Pastikan task1 dan task2 tetap di todo (tidak diubah)
	_ = task1
	_ = task2

	t.Run("filter todo → 2 task", func(t *testing.T) {
		tasks, err := svc.GetAll("todo")
		if err != nil {
			t.Fatalf("GetAll(todo) error = %v", err)
		}
		if len(tasks) != 2 {
			t.Errorf("GetAll(todo) = %d task, want 2", len(tasks))
		}
		for _, task := range tasks {
			if task.Status != model.StatusTodo {
				t.Errorf("GetAll(todo) mengembalikan status %q, want todo", task.Status)
			}
		}
	})

	t.Run("filter in_progress → 1 task", func(t *testing.T) {
		tasks, err := svc.GetAll("in_progress")
		if err != nil {
			t.Fatalf("GetAll(in_progress) error = %v", err)
		}
		if len(tasks) != 1 {
			t.Errorf("GetAll(in_progress) = %d task, want 1", len(tasks))
		}
		if len(tasks) > 0 && tasks[0].Status != model.StatusInProgress {
			t.Errorf("GetAll(in_progress) mengembalikan status %q", tasks[0].Status)
		}
	})

	t.Run("filter done → 1 task", func(t *testing.T) {
		tasks, err := svc.GetAll("done")
		if err != nil {
			t.Fatalf("GetAll(done) error = %v", err)
		}
		if len(tasks) != 1 {
			t.Errorf("GetAll(done) = %d task, want 1", len(tasks))
		}
		if len(tasks) > 0 && tasks[0].Status != model.StatusDone {
			t.Errorf("GetAll(done) mengembalikan status %q", tasks[0].Status)
		}
	})

	t.Run("filter kosong → semua task (4)", func(t *testing.T) {
		tasks, err := svc.GetAll("")
		if err != nil {
			t.Fatalf("GetAll('') error = %v", err)
		}
		if len(tasks) != 4 {
			t.Errorf("GetAll('') = %d task, want 4", len(tasks))
		}
	})

	t.Run("filter invalid → error", func(t *testing.T) {
		_, err := svc.GetAll("cancelled")
		if err == nil {
			t.Error("GetAll('cancelled') harus error untuk status tidak valid")
		}
	})
}

// ── TestGetStats_CompletionRate ─────────────────────────────────────────────

func TestGetStats_CompletionRate(t *testing.T) {
	t.Run("0% — tidak ada task done", func(t *testing.T) {
		svc := newSvc()
		svc.Create(model.CreateTaskRequest{Title: "T1"}) //nolint
		svc.Create(model.CreateTaskRequest{Title: "T2"}) //nolint

		stats, err := svc.GetStats()
		if err != nil {
			t.Fatalf("GetStats() error = %v", err)
		}
		if stats.CompletionRate != 0 {
			t.Errorf("CompletionRate = %.2f, want 0.00", stats.CompletionRate)
		}
	})

	t.Run("50% — setengah task done", func(t *testing.T) {
		svc := newSvc()
		task1, _ := svc.Create(model.CreateTaskRequest{Title: "T1"})
		svc.Create(model.CreateTaskRequest{Title: "T2"}) //nolint

		done := model.StatusDone
		svc.Update(task1.ID, model.UpdateTaskRequest{Status: &done}) //nolint

		stats, err := svc.GetStats()
		if err != nil {
			t.Fatalf("GetStats() error = %v", err)
		}
		diff := stats.CompletionRate - 50.0
		if diff < 0 {
			diff = -diff
		}
		if diff > 0.01 {
			t.Errorf("CompletionRate = %.2f, want 50.00", stats.CompletionRate)
		}
	})

	t.Run("33.33% — sepertiga task done", func(t *testing.T) {
		svc := newSvc()
		task1, _ := svc.Create(model.CreateTaskRequest{Title: "T1"})
		svc.Create(model.CreateTaskRequest{Title: "T2"}) //nolint
		svc.Create(model.CreateTaskRequest{Title: "T3"}) //nolint

		done := model.StatusDone
		svc.Update(task1.ID, model.UpdateTaskRequest{Status: &done}) //nolint

		stats, err := svc.GetStats()
		if err != nil {
			t.Fatalf("GetStats() error = %v", err)
		}
		diff := stats.CompletionRate - 33.33
		if diff < 0 {
			diff = -diff
		}
		if diff > 0.01 {
			t.Errorf("CompletionRate = %.2f, want 33.33", stats.CompletionRate)
		}
	})

	t.Run("100% — semua task done", func(t *testing.T) {
		svc := newSvc()
		task1, _ := svc.Create(model.CreateTaskRequest{Title: "T1"})
		task2, _ := svc.Create(model.CreateTaskRequest{Title: "T2"})

		done := model.StatusDone
		svc.Update(task1.ID, model.UpdateTaskRequest{Status: &done}) //nolint
		svc.Update(task2.ID, model.UpdateTaskRequest{Status: &done}) //nolint

		stats, err := svc.GetStats()
		if err != nil {
			t.Fatalf("GetStats() error = %v", err)
		}
		if stats.CompletionRate != 100.0 {
			t.Errorf("CompletionRate = %.2f, want 100.00", stats.CompletionRate)
		}
	})

	t.Run("0% — tidak ada task sama sekali", func(t *testing.T) {
		svc := newSvc()

		stats, err := svc.GetStats()
		if err != nil {
			t.Fatalf("GetStats() error = %v", err)
		}
		if stats.CompletionRate != 0 {
			t.Errorf("CompletionRate = %.2f, want 0.00", stats.CompletionRate)
		}
		if stats.Total != 0 {
			t.Errorf("Total = %d, want 0", stats.Total)
		}
	})
}

// ── [TODO] Semua test sudah diimplementasikan ─────────────────────────────────
// ✅ TestGetAll_WithStatusFilter
// ✅ TestGetStats_CompletionRate
// ✅ TestCreate_WithUnicodeTitle
// ✅ TestDelete_AndVerifyStats
