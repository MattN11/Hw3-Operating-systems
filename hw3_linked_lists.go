package main

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// Node represents a node in the linked list
type Node struct {
	Value int
	Next  *Node
}

// IMPLEMENTATION 1: Coarse-Grained Locking (Figure 29.8)
// Single lock protecting the entire list - simple but less concurrent

// CoarseGrainedList uses a single mutex to protect the entire list
type CoarseGrainedList struct {
	mu   sync.Mutex
	head *Node
}

// NewCoarseGrainedList creates a new coarse-grained list
func NewCoarseGrainedList() *CoarseGrainedList {
	return &CoarseGrainedList{
		head: nil,
	}
}

// Insert adds a value to the list in sorted order
func (l *CoarseGrainedList) Insert(value int) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Check if value already exists
	current := l.head
	var prev *Node
	for current != nil && current.Value < value {
		prev = current
		current = current.Next
	}

	if current != nil && current.Value == value {
		return false // Value already exists
	}

	newNode := &Node{Value: value}
	newNode.Next = current

	if prev == nil {
		l.head = newNode
	} else {
		prev.Next = newNode
	}

	return true
}

// Delete removes a value from the list
func (l *CoarseGrainedList) Delete(value int) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.head == nil {
		return false
	}

	if l.head.Value == value {
		l.head = l.head.Next
		return true
	}

	current := l.head
	for current.Next != nil && current.Next.Value < value {
		current = current.Next
	}

	if current.Next != nil && current.Next.Value == value {
		current.Next = current.Next.Next
		return true
	}

	return false
}

// Search checks if a value exists in the list
func (l *CoarseGrainedList) Search(value int) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	current := l.head
	for current != nil {
		if current.Value == value {
			return true
		}
		if current.Value > value {
			return false
		}
		current = current.Next
	}
	return false
}

// ========== IMPLEMENTATION 2: Hand-Over-Hand Locking (Optimized) ==========
// Each node has its own lock; acquire locks sequentially as we traverse

// LockableNode wraps a node with its own lock
type LockableNode struct {
	Value int
	Next  *LockableNode
	mu    sync.Mutex
}

// FineGrainedList uses hand-over-hand locking for better concurrency
type FineGrainedList struct {
	mu   sync.Mutex // protects head pointer only
	head *LockableNode
}

// NewFineGrainedList creates a new fine-grained list
func NewFineGrainedList() *FineGrainedList {
	return &FineGrainedList{
		head: nil,
	}
}

// Insert adds a value using hand-over-hand locking
func (l *FineGrainedList) Insert(value int) bool {
	l.mu.Lock()
	current := l.head

	if current == nil {
		l.head = &LockableNode{Value: value}
		l.mu.Unlock()
		return true
	}

	current.mu.Lock()
	l.mu.Unlock()

	// Hand-over-hand: traverse while holding current lock, then acquire next
	for current.Next != nil && current.Next.Value < value {
		next := current.Next
		next.mu.Lock()
		current.mu.Unlock()
		current = next
	}

	defer current.mu.Unlock()

	// Check if value already exists
	if current.Value == value {
		return false
	}

	if current.Next != nil && current.Next.Value == value {
		return false
	}

	// Insert after current
	newNode := &LockableNode{Value: value}
	newNode.Next = current.Next
	current.Next = newNode
	return true
}

// Delete removes a value using hand-over-hand locking
func (l *FineGrainedList) Delete(value int) bool {
	l.mu.Lock()
	current := l.head

	if current == nil {
		l.mu.Unlock()
		return false
	}

	if current.Value == value {
		l.head = current.Next
		l.mu.Unlock()
		return true
	}

	current.mu.Lock()
	l.mu.Unlock()

	for current.Next != nil {
		if current.Next.Value == value {
			next := current.Next
			next.mu.Lock()
			current.Next = next.Next
			next.mu.Unlock()
			current.mu.Unlock()
			return true
		}

		if current.Next.Value > value {
			current.mu.Unlock()
			return false
		}

		next := current.Next
		next.mu.Lock()
		current.mu.Unlock()
		current = next
	}

	current.mu.Unlock()
	return false
}

// Search checks if a value exists using hand-over-hand locking
func (l *FineGrainedList) Search(value int) bool {
	l.mu.Lock()
	current := l.head

	if current == nil {
		l.mu.Unlock()
		return false
	}

	current.mu.Lock()
	l.mu.Unlock()

	for current != nil {
		if current.Value == value {
			current.mu.Unlock()
			return true
		}

		if current.Value > value {
			current.mu.Unlock()
			return false
		}

		if current.Next == nil {
			current.mu.Unlock()
			return false
		}

		next := current.Next
		next.mu.Lock()
		current.mu.Unlock()
		current = next
	}

	return false
}

// ========== BENCHMARKING UTILITIES ==========

// ListOps interface for common operations
type ListOps interface {
	Insert(value int) bool
	Delete(value int) bool
	Search(value int) bool
}

// BenchmarkResult holds the results of a benchmark
type BenchmarkResult struct {
	Name          string
	NumGoroutines int
	NumOperations int
	Duration      time.Duration
	ThroughputOps float64 // operations per second
}

// Workload1: Heavy inserts (60% inserts, 20% deletes, 20% searches)
func workload1(list ListOps, value int, op int) {
	if op%10 < 6 {
		list.Insert(value)
	} else if op%10 < 8 {
		list.Delete(value)
	} else {
		list.Search(value)
	}
}

// Workload2: Read-heavy (10% inserts, 10% deletes, 80% searches)
func workload2(list ListOps, value int, op int) {
	if op%10 < 1 {
		list.Insert(value)
	} else if op%10 < 2 {
		list.Delete(value)
	} else {
		list.Search(value)
	}
}

// Workload3: Write-heavy (40% inserts, 40% deletes, 20% searches)
func workload3(list ListOps, value int, op int) {
	if op%10 < 4 {
		list.Insert(value)
	} else if op%10 < 8 {
		list.Delete(value)
	} else {
		list.Search(value)
	}
}

// RunBenchmark executes a benchmark for a given list implementation
func RunBenchmark(name string, list ListOps, numGoroutines int, numOpsPerGoroutine int, workloadFunc func(ListOps, int, int)) BenchmarkResult {
	start := time.Now()

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < numOpsPerGoroutine; j++ {
				value := (goroutineID*numOpsPerGoroutine + j) % 1000
				op := (goroutineID*numOpsPerGoroutine + j) % 10
				workloadFunc(list, value, op)
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)

	totalOps := numGoroutines * numOpsPerGoroutine
	throughput := float64(totalOps) / duration.Seconds()

	return BenchmarkResult{
		Name:          name,
		NumGoroutines: numGoroutines,
		NumOperations: totalOps,
		Duration:      duration,
		ThroughputOps: throughput,
	}
}

// ========== MAIN BENCHMARKING SUITE ==========

func runManualBenchmarks() {
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("CONCURRENT LINKED LIST BENCHMARKING RESULTS")
	fmt.Println(strings.Repeat("=", 80) + "\n")

	workloads := []struct {
		name string
		fn   func(ListOps, int, int)
		desc string
	}{
		{"Workload1_InsertHeavy", workload1, "Heavy Insert (60% insert, 20% delete, 20% search)"},
		{"Workload2_ReadHeavy", workload2, "Read-Heavy (10% insert, 10% delete, 80% search)"},
		{"Workload3_WriteHeavy", workload3, "Write-Heavy (40% insert, 40% delete, 20% search)"},
	}

	goroutineConfigs := []int{1, 2, 4, 8, 16}

	for _, config := range goroutineConfigs {
		fmt.Printf("\n%s\n", strings.Repeat("=", 80))
		fmt.Printf("Number of Goroutines: %d\n", config)
		fmt.Printf("%s\n\n", strings.Repeat("=", 80))

		for _, workload := range workloads {
			fmt.Printf("### %s ###\n%s\n", workload.name, workload.desc)

			// Pre-populate lists with initial values
			coarseList := NewCoarseGrainedList()
			fineList := NewFineGrainedList()

			for i := 0; i < 100; i++ {
				coarseList.Insert(i)
				fineList.Insert(i)
			}

			opsPerGoroutine := 10000

			coarseResult := RunBenchmark("Coarse-Grained", coarseList, config, opsPerGoroutine, workload.fn)
			fineResult := RunBenchmark("Fine-Grained", fineList, config, opsPerGoroutine, workload.fn)

			fmt.Printf("\nCoarse-Grained Locking:\n")
			fmt.Printf("  Total Operations: %d\n", coarseResult.NumOperations)
			fmt.Printf("  Duration: %v\n", coarseResult.Duration)
			fmt.Printf("  Throughput: %.2f ops/sec\n", coarseResult.ThroughputOps)

			fmt.Printf("\nFine-Grained (Hand-over-Hand) Locking:\n")
			fmt.Printf("  Total Operations: %d\n", fineResult.NumOperations)
			fmt.Printf("  Duration: %v\n", fineResult.Duration)
			fmt.Printf("  Throughput: %.2f ops/sec\n", fineResult.ThroughputOps)

			improvement := (fineResult.ThroughputOps - coarseResult.ThroughputOps) / coarseResult.ThroughputOps * 100
			fmt.Printf("\nPerformance Difference: %.2f%% ", improvement)
			if improvement > 0 {
				fmt.Printf("(Fine-Grained is FASTER)\n")
			} else {
				fmt.Printf("(Coarse-Grained is FASTER)\n")
			}
			fmt.Printf("%s\n\n", strings.Repeat("-", 80))
		}
	}
}

func main() {
	runManualBenchmarks()
}
