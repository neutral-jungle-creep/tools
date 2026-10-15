package background

import (
	"context"
	"errors"
	"sync"
)

type Job func() error

type StopHandler func() error

// Регистрация безопасна из нескольких горутин; Run и Stop вызываются по одному разу.
type Registrar struct {
	mu    sync.Mutex
	jobs  []Job
	stops []StopHandler
}

func New() *Registrar {
	return &Registrar{}
}

func (r *Registrar) RegisterJob(j Job) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.jobs = append(r.jobs, j)
}

func (r *Registrar) RegisterStopHandler(h StopHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.stops = append(r.stops, h)
}

// Run не вызывает Stop: время освобождения ресурсов определяет вызывающий код.
func (r *Registrar) Run(ctx context.Context) error {
	r.mu.Lock()
	jobs := append([]Job(nil), r.jobs...)
	r.mu.Unlock()

	// Буфер не даёт горутинам зависнуть после возврата Run по отмене контекста.
	errs := make(chan error, len(jobs))
	for _, j := range jobs {
		go func(job Job) {
			errs <- job()
		}(j)
	}

	select {
	case <-ctx.Done():
		return nil
	case err := <-errs:
		return err
	}
}

// Stop ждёт все обработчики и объединяет их ошибки.
func (r *Registrar) Stop() error {
	r.mu.Lock()
	stops := append([]StopHandler(nil), r.stops...)
	r.mu.Unlock()

	errs := make([]error, len(stops))
	var wg sync.WaitGroup
	wg.Add(len(stops))
	for i, h := range stops {
		go func(idx int, handler StopHandler) {
			defer wg.Done()
			errs[idx] = handler()
		}(i, h)
	}
	wg.Wait()

	return errors.Join(errs...)
}
