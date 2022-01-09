package main

import (
	"context"
	"log"
	"os"

	"github.com/cschleiden/go-dt/pkg/backend"
	"github.com/cschleiden/go-dt/pkg/backend/sqlite"
	"github.com/cschleiden/go-dt/pkg/client"
	"github.com/cschleiden/go-dt/pkg/worker"
	"github.com/cschleiden/go-dt/pkg/workflow"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

func main() {
	ctx := context.Background()

	//b := memory.NewMemoryBackend()
	b := sqlite.NewSqliteBackend("subworkflow.sqlite")

	// Run worker
	go RunWorker(ctx, b)

	// Start workflow via client
	c := client.NewClient(b)

	startWorkflow(ctx, c)

	c2 := make(chan os.Signal, 1)
	<-c2
}

func startWorkflow(ctx context.Context, c client.Client) {
	wf, err := c.CreateWorkflowInstance(ctx, client.WorkflowInstanceOptions{
		InstanceID: uuid.NewString(),
	}, Workflow1, "Hello world"+uuid.NewString())
	if err != nil {
		panic("could not start workflow")
	}

	log.Println("Started workflow", wf.GetInstanceID())
}

func RunWorker(ctx context.Context, mb backend.Backend) {
	w := worker.NewWorker(mb)

	w.RegisterWorkflow("wf1", Workflow1)
	w.RegisterWorkflow("wf2", Workflow2)

	w.RegisterActivity("a1", Activity1)
	w.RegisterActivity("a2", Activity2)

	if err := w.Start(ctx); err != nil {
		panic("could not start worker")
	}
}

func Workflow1(ctx workflow.Context, msg string) error {
	log.Println("Entering Workflow1")
	log.Println("\tWorkflow instance input:", msg)
	log.Println("\tIsReplaying:", workflow.Replaying(ctx))

	defer func() {
		log.Println("Leaving Workflow1")
	}()

	w2, err := workflow.CreateSubWorkflowInstance(ctx, "wf2", "some input")
	if err != nil {
		return errors.Wrap(err, "failed to create sub workflow")
	}

	var wr string
	if err := w2.Get(ctx, &wr); err != nil {
		return errors.Wrap(err, "could not get sub workflow result")
	}

	log.Println("Sub workflow result:", wr)

	return nil
}

func Workflow2(ctx workflow.Context, msg string) (string, error) {
	log.Println("Entering Workflow2")
	log.Println("\tWorkflow instance input:", msg)
	log.Println("\tIsReplaying:", workflow.Replaying(ctx))

	defer func() {
		log.Println("Leaving Workflow2")
	}()

	a1, err := workflow.ExecuteActivity(ctx, "a1", 35, 12)
	if err != nil {
		panic("error executing activity 1")
	}

	var r1 int
	err = a1.Get(ctx, &r1)
	if err != nil {
		panic("error getting activity 1 result")
	}
	log.Println("R1 result:", r1)
	log.Println("\tIsReplaying:", workflow.Replaying(ctx))

	a2, err := workflow.ExecuteActivity(ctx, "a2")
	if err != nil {
		panic("error executing activity 1")
	}

	var r2 int
	err = a2.Get(ctx, &r2)
	if err != nil {
		panic("error getting activity 1 result")
	}
	log.Println("R2 result:", r2)
	log.Println("\tIsReplaying:", workflow.Replaying(ctx))

	return "W2 Result", nil
}

func Activity1(ctx context.Context, a, b int) (int, error) {
	log.Println("Entering Activity1")

	defer func() {
		log.Println("Leaving Activity1")
	}()

	return a + b, nil
}

func Activity2(ctx context.Context) (int, error) {
	log.Println("Entering Activity2")

	defer func() {
		log.Println("Leaving Activity2")
	}()

	return 12, nil
}