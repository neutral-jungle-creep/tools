package background_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/neutral-jungle-creep/tools/background"
)

func TestRegistrar_Run_NoJobsBlocksUntilCtxCancel(t *testing.T) {
	t.Parallel()

	r := background.New()

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- r.Run(ctx) }()

	select {
	case <-done:
		t.Fatal("Run returned before ctx was cancelled")
	case <-time.After(50 * time.Millisecond):
	}

	cancel()
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("Run did not return after ctx cancel")
	}
}

func TestRegistrar_Run_ReturnsFirstJobError(t *testing.T) {
	t.Parallel()

	r := background.New()
	r.RegisterJob(func() error {
		time.Sleep(20 * time.Millisecond)
		return nil
	})
	want := errors.New("kaboom")
	r.RegisterJob(func() error { return want })

	err := r.Run(context.Background())
	require.ErrorIs(t, err, want)
}

func TestRegistrar_Run_CtxCancelWinsOverPendingJobs(t *testing.T) {
	t.Parallel()

	r := background.New()
	r.RegisterJob(func() error {
		time.Sleep(time.Second)
		return errors.New("never")
	})

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	err := r.Run(ctx)
	require.NoError(t, err, "ctx cancel should beat the slow job")
}

func TestRegistrar_Stop_RunsHandlersInReverseOrder(t *testing.T) {
	t.Parallel()

	r := background.New()
	var order []int
	for i := 0; i < 3; i++ {
		index := i
		r.RegisterStopHandler(func() error {
			order = append(order, index)
			return nil
		})
	}

	require.NoError(t, r.Stop())
	assert.Equal(t, []int{2, 1, 0}, order)
}

func TestRegistrar_Stop_NoHandlersIsNoOp(t *testing.T) {
	t.Parallel()

	r := background.New()
	assert.NotPanics(t, func() { _ = r.Stop() })
}

func TestRegistrar_Stop_JoinsErrors(t *testing.T) {
	t.Parallel()

	r := background.New()
	errA := errors.New("close A failed")
	errB := errors.New("close B failed")
	var order []string
	r.RegisterStopHandler(func() error {
		order = append(order, "A")
		return errA
	})
	r.RegisterStopHandler(func() error {
		order = append(order, "middle")
		return nil
	})
	r.RegisterStopHandler(func() error {
		order = append(order, "B")
		return errB
	})

	err := r.Stop()
	require.ErrorIs(t, err, errA)
	require.ErrorIs(t, err, errB)
	assert.Equal(t, []string{"B", "middle", "A"}, order)
}

func TestRegistrar_ConcurrentRegistrationIsSafe(t *testing.T) {
	t.Parallel()

	r := background.New()
	var wg sync.WaitGroup
	const goroutines = 20
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			r.RegisterJob(func() error { return nil })
			r.RegisterStopHandler(func() error { return nil })
		}()
	}
	wg.Wait()

	_ = r.Stop()
}
