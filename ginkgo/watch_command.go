package main

import (
	"flag"
	"fmt"
	"github.com/onsi/ginkgo/ginkgo/testrunner"
	"github.com/onsi/ginkgo/ginkgo/testsuite"
)

func BuildWatchCommand() *Command {
	commandFlags := NewWatchCommandFlags(flag.NewFlagSet("watch", flag.ExitOnError))
	watcher := &SpecWatcher{
		commandFlags:     commandFlags,
		notifier:         NewNotifier(commandFlags),
		interruptHandler: NewInterruptHandler(),
	}

	return &Command{
		Name:         "watch",
		FlagSet:      commandFlags.FlagSet,
		UsageCommand: "ginkgo watch <FLAGS> <PACKAGES>...",
		Usage: []string{
			"Watches the tests in the passed in <PACKAGES> and runs them when changes occur.",
		},
		Command:                   watcher.WatchSpecs,
		SuppressFlagDocumentation: true,
		FlagDocSubstitute: []string{
			"Accepts all the flags that the ginkgo command accepts except for --keepGoing and --untilItFails",
		},
	}
}

type SpecWatcher struct {
	commandFlags     *RunAndWatchCommandFlags
	notifier         *Notifier
	interruptHandler *InterruptHandler
}

func (w *SpecWatcher) WatchSpecs(args []string) {
	w.notifier.VerifyNotificationsAreAvailable()

	suites := findSuites(args, w.commandFlags.Recurse, w.commandFlags.SkipPackage)
	w.WatchSuites(suites)
}

func (w *SpecWatcher) WatchSuites(suites []*testsuite.TestSuite) {
	modifiedSuite := make(chan *testsuite.TestSuite)
	for _, suite := range suites {
		go suite.Watch(modifiedSuite)
	}

	if len(suites) == 1 {
		w.RunSuite(suites[0])
	}

	for {
		select {
		case suite := <-modifiedSuite:
			w.notifier.SendNotification("Ginkgo", fmt.Sprintf(`Detected change in "%s"...`, suite.PackageName))

			fmt.Printf("\n\nDetected change in %s\n\n", suite.PackageName)
			w.RunSuite(suite)
		case <-w.interruptHandler.C:
			return
		}
	}
}

func (w *SpecWatcher) RunSuite(suite *testsuite.TestSuite) {
	runner := testrunner.New(suite, w.commandFlags.NumCPU, w.commandFlags.ParallelStream, w.commandFlags.Race, w.commandFlags.Cover)
	err := runner.Compile()
	if err != nil {
		fmt.Print(err.Error())
	}
	suitePassed := (err == nil) && runner.Run()
	w.notifier.SendSuiteCompletionNotification(suite, suitePassed)
	runner.CleanUp()
}
