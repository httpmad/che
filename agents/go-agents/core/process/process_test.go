//
// Copyright (c) 2012-2017 Codenvy, S.A.
// All rights reserved. This program and the accompanying materials
// are made available under the terms of the Eclipse Public License v1.0
// which accompanies this distribution, and is available at
// http://www.eclipse.org/legal/epl-v10.html
//
// Contributors:
//   Codenvy, S.A. - initial API and implementation
//

package process_test

import (
	"log"
	"math/rand"
	"os"
	"strings"
	"testing"
	"time"

	"fmt"
	"github.com/eclipse/che/agents/go-agents/core/process"
	"github.com/eclipse/che/agents/go-agents/core/rpc"
	"sync"
)

const (
	testCmd = "printf \"1\n2\n3\n4\n5\n6\n7\n8\n9\n10\""
)

var alphabet = []byte("abcdefgh123456789")

func TestOneLineOutput(t *testing.T) {
	defer wipeLogs()
	// create and start a process
	p := startAndWaitTestProcessWritingLogsToTmpDir("echo test", t)

	logs, _ := process.ReadAllLogs(p.Pid)

	if len(logs) != 1 {
		t.Fatalf("Expected logs size to be 1, but got %d", len(logs))
	}

	if logs[0].Text != "test" {
		t.Fatalf("Expected to get 'test' output but got %s", logs[0].Text)
	}
}

func TestEmptyLinesOutput(t *testing.T) {
	p := startAndWaitTestProcessWritingLogsToTmpDir("printf \"\n\n\n\n\n\"", t)
	defer process.WipeLogs()

	logs, _ := process.ReadAllLogs(p.Pid)

	if len(logs) != 5 {
		t.Fatalf("Expected logs to be 5 sized, but the size is '%d'", len(logs))
	}

	for _, value := range logs {
		if value.Text != "" {
			t.Fatal("Expected all the logs to be empty files")
		}
	}
}

func TestAddSubscriber(t *testing.T) {
	outputLines := []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10"}

	// create and start a process
	pb := process.NewBuilder()
	pb.CmdName("test")
	pb.CmdType("test")
	pb.CmdLine("printf \"" + strings.Join(outputLines, "\n") + "\"")

	// add a new subscriber
	eventsChan := make(chan *rpc.Event)
	pb.SubscribeDefault("test", eventsChan)

	// start a new process
	if _, err := pb.Start(); err != nil {
		t.Fatal(err)
	}

	// read all the process output events
	done := make(chan bool)
	var received []string
	go func() {
		event := <-eventsChan
		for event.EventType != process.DiedEventType {
			if event.EventType == process.StdoutEventType {
				out := event.Body.(*process.OutputEventBody)
				received = append(received, out.Text)
			}
			event = <-eventsChan
		}
		done <- true
	}()

	// wait until process is done
	<-done

	if len(outputLines) != len(received) {
		t.Fatalf("Expected the same size but got %d != %d", len(outputLines), len(received))
	}

	for idx, value := range outputLines {
		if value != received[idx] {
			t.Fatalf("Expected %s but got %s", value, received[idx])
		}
	}
}

func TestRestoreSubscriberForDeadProcess(t *testing.T) {
	beforeStart := time.Now()
	p := startAndWaitTestProcessWritingLogsToTmpDir("echo test", t)
	defer process.WipeLogs()

	// Read all the data from channel
	channel := make(chan *rpc.Event)
	done := make(chan bool)
	var received []*rpc.Event
	go func() {
		statusReceived := false
		timeoutReached := false
		for !statusReceived && !timeoutReached {
			select {
			case v := <-channel:
				received = append(received, v)
				if v.EventType == process.DiedEventType {
					statusReceived = true
				}
			case <-time.After(time.Second):
				timeoutReached = true
			}
		}
		done <- true
	}()

	_ = process.RestoreSubscriber(p.Pid, process.Subscriber{
		ID:      "test",
		Mask:    process.DefaultMask,
		Channel: channel,
	}, beforeStart)

	<-done

	if len(received) != 2 {
		t.Fatalf("Expected to recieve 2 events but got %d", len(received))
	}
	e1Type := received[0].EventType
	e1Text := received[0].Body.(*process.OutputEventBody).Text
	if received[0].EventType != process.StdoutEventType || e1Text != "test" {
		t.Fatalf("Expected to receieve output event with text 'test', but got '%s' event with text %s",
			e1Type,
			e1Text)
	}
	if received[1].EventType != process.DiedEventType {
		t.Fatal("Expected to get 'process_died' event")
	}
}

func TestMachineProcessIsNotAliveAfterItIsDead(t *testing.T) {
	p := startAndWaitTestProcess(testCmd, t)
	if p.Alive {
		t.Fatal("Process should not be alive")
	}
}

func TestItIsNotPossibleToAddSubscriberToDeadProcess(t *testing.T) {
	p := startAndWaitTestProcess(testCmd, t)
	if err := process.AddSubscriber(p.Pid, process.Subscriber{}); err == nil {
		t.Fatal("Should not be able to add subscriber")
	}
}

func TestReadProcessLogs(t *testing.T) {
	p := startAndWaitTestProcessWritingLogsToTmpDir(testCmd, t)
	defer wipeLogs()
	logs, err := process.ReadLogs(p.Pid, time.Time{}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10"}

	for idx := range expected {
		if process.StdoutKind != logs[idx].Kind {
			t.Fatalf("Expected log message kind to be '%s', while got '%s'", process.StdoutKind, logs[idx].Kind)
		}
		if expected[idx] != logs[idx].Text {
			t.Fatalf("Expected log message to be '%s', but got '%s'", expected[idx], logs[idx].Text)
		}
	}
}

func TestLogsAreNotWrittenIfLogsDirIsNotSet(t *testing.T) {
	p := doStartAndWaitTestProcess(testCmd, "", &eventsCaptor{deathEventType: process.DiedEventType}, t)

	_, err := process.ReadAllLogs(p.Pid)
	if err == nil {
		t.Fatal("Error must be returned in the case when the process doesn't write logs")
	}

	expected := fmt.Sprintf("Logs file for process '%d' is missing", p.Pid)
	if err.Error() != expected {
		t.Fatalf("Expected to get '%s' error but got '%s'", err.Error(), expected)
	}
}

func TestAllProcessLifeCycleEventsArePublished(t *testing.T) {
	eventsCaptor := &eventsCaptor{deathEventType: process.DiedEventType}
	doStartAndWaitTestProcess("printf \"first_line\nsecond_line\"", "", eventsCaptor, t)

	expected := []string{
		process.StartedEventType,
		process.StdoutEventType,
		process.StdoutEventType,
		process.DiedEventType,
	}
	checkEventsOrder(t, eventsCaptor.events, expected...)
}

func TestProcessExitCodeIs0IfFinishedOk(t *testing.T) {
	captor := &eventsCaptor{deathEventType: process.DiedEventType}
	p := doStartAndWaitTestProcess("echo test", "", captor, t)

	if p.ExitCode != 0 {
		t.Fatalf("Expected process exit code to be 0, but it is %d", p.ExitCode)
	}

	diedEvent := captor.events[len(captor.events)-1]
	if body, ok := diedEvent.Body.(*process.DiedEventBody); !ok {
		t.Fatalf("Expected last captured event to be process died event, but it is %s", diedEvent.EventType)
	} else if body.ExitCode != 0 {
		t.Fatalf("Expected process died event exit code to be 0, but it is %d", body.ExitCode)
	}
}

func TestProcessExitCodeIsNot0IfFinishedNotOk(t *testing.T) {
	captor := &eventsCaptor{deathEventType: process.DiedEventType}
	// starting non-existing command(hopefully)
	p := doStartAndWaitTestProcess("test-process-cmd-"+randomName(10), "", captor, t)

	if p.ExitCode <= 0 {
		t.Fatalf("Expected process exit code to be > 0, but it is %d", p.ExitCode)
	}

	diedEvent := captor.events[len(captor.events)-1]
	if body, ok := diedEvent.Body.(*process.DiedEventBody); !ok {
		t.Fatalf("Expected last captured event to be process died event, but it is %s", diedEvent.EventType)
	} else if body.ExitCode <= 0 {
		t.Fatalf("Expected process died event exit code to be > 0, but it is %d", body.ExitCode)
	}
}

func checkEventsOrder(t *testing.T, events []*rpc.Event, types ...string) {
	if len(types) != len(events) {
		t.Fatalf("Expected receive %d events while received %d", len(types), len(events))
	}
	for idx := range types {
		failIfEventTypeIsDifferent(t, events[idx], types[idx])
	}
}

func failIfEventTypeIsDifferent(t *testing.T, event *rpc.Event, expectedType string) {
	if event.EventType != expectedType {
		t.Fatalf("Expected event type to be '%s' but it is '%s'", expectedType, event.EventType)
	}
}

func startAndWaitTestProcess(cmd string, t *testing.T) process.MachineProcess {
	p := doStartAndWaitTestProcess(cmd, "", &eventsCaptor{deathEventType: process.DiedEventType}, t)
	return p
}

func startAndWaitTestProcessWritingLogsToTmpDir(cmd string, t *testing.T) process.MachineProcess {
	p := doStartAndWaitTestProcess(cmd, tmpFile(), &eventsCaptor{deathEventType: process.DiedEventType}, t)
	return p
}

func doStartAndWaitTestProcess(cmd string, logsDir string, eventsCaptor *eventsCaptor, t *testing.T) process.MachineProcess {
	process.SetLogsDir(logsDir)

	eventsCaptor.capture()

	pb := process.NewBuilder()
	pb.CmdName("test")
	pb.CmdType("test")
	pb.CmdLine(cmd)
	pb.SubscribeDefault("events-captor", eventsCaptor.eventsChan)

	p, err := pb.Start()
	if err != nil {
		eventsCaptor.wait(0)
		t.Fatal(err)
	}

	// wait process for a little while
	if ok := <-eventsCaptor.wait(time.Second * 2); !ok {
		t.Log("The process doesn't finish its execution in 2 seconds. Trying to kill it")
		if err := process.Kill(p.Pid); err != nil {
			t.Logf("Failed to kill process, native pid = %d", p.NativePid)
		}
		t.FailNow()
	}

	// check process state after it is finished
	result, err := process.Get(p.Pid)
	if err != nil {
		t.Fatal(err)
	}

	return result
}

func tmpFile() string {
	return os.TempDir() + string(os.PathSeparator) + randomName(10)
}

func wipeLogs() {
	if err := process.WipeLogs(); err != nil {
		log.Printf("Could not wipe process logs dir. %s", err.Error())
	}
}

func randomName(length int) string {
	rand.Seed(time.Now().UnixNano())
	bytes := make([]byte, length)
	for i := 0; i < length; i++ {
		bytes[i] = alphabet[rand.Intn(len(alphabet))]
	}
	return string(bytes)
}

// Helps to capture process events and wait for them.
type eventsCaptor struct {
	sync.Mutex

	// Result events.
	events []*rpc.Event

	// Events channel. Close of this channel considered as immediate interruption,
	// to hold until execution completes use captor.wait(timeout) channel.
	eventsChan chan *rpc.Event

	// Channel used as internal approach to interrupt capturing.
	interruptChan chan bool

	// Captor sends true if finishes reaching deathEventType
	// and false if interrupted while waiting for event of deathEventType.
	done chan bool

	// The last event after which events capturing stopped.
	deathEventType string
}

func (ec *eventsCaptor) addEvent(e *rpc.Event) {
	ec.Lock()
	defer ec.Unlock()
	ec.events = append(ec.events, e)
}

func (ec *eventsCaptor) capturedEvents() []*rpc.Event {
	ec.Lock()
	defer ec.Unlock()
	cp := make([]*rpc.Event, len(ec.events))
	copy(cp, ec.events)
	return cp
}

func (ec *eventsCaptor) capture() {
	ec.eventsChan = make(chan *rpc.Event)
	ec.interruptChan = make(chan bool)
	ec.done = make(chan bool)

	go func() {
		for {
			select {
			case event, ok := <-ec.eventsChan:
				if ok {
					ec.addEvent(event)
					if event.EventType == ec.deathEventType {
						// death event reached - capturing is done
						ec.done <- true
						return
					}
				} else {
					// events channel closed interrupt immediately
					ec.done <- false
					return
				}
			case <-ec.interruptChan:
				close(ec.eventsChan)
			}
		}
	}()
}

// Waits a timeout and if deadTypeEvent wasn't reached interrupts captor.
func (ec *eventsCaptor) wait(timeout time.Duration) chan bool {
	go func() {
		<-time.NewTimer(timeout).C
		ec.interruptChan <- true
	}()
	return ec.done
}