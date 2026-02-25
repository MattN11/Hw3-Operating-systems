# Hw3-Operating-systems
# HW3: Go Concurrency & Linked Lists 
Q2


everything's in `hw3_linked_lists.go`. that's the file. that's what runs. boom.

---

## what the code actually does

### way: `CoarseGrainedList`

you throw one big lock on the whole list. wanna add something? lock it. 
```go
type CoarseGrainedList struct {
    head *Node
    mu   sync.Mutex
}
```

so you:
1. grab the lock
2. do your thing
3. let go of the lock
4. done

works great if only one thread. but if like 10 threads all try at once? yeah they all gotta wait for each other lol

---

### the fancy way: `FineGrainedList`

every node gets its own lock. so you're walking through the list like:
1. lock node A
2. look at the next node (B)
3. lock B
4. let go of A
5. now you're on B, repeat

it's like passing the baton. annoying to code, but more threads can actually do stuff at the same time. theoretically.

```go
type LockableNode struct {
    value int
    next  *LockableNode
    mu    sync.Mutex
}
```

---

## what happens when you run it

ok so:

**step 1:** makes two lists, fills each with like 100 nodes

**step 2:** runs 3 different tests, and for each test it tries with 1, 2, 4, 8, and 16 threads. each thread just hammers the list with 10k random operations.

**step 3:** the three tests are:

**test 1: lots of inserts** (60% insert, 20% delete, 20% search) - see if adding stuff breaks things

**test 2: lots of searches** (80% search, 10% insert, 10% delete) - see if the lock even matters when you're not changing anything

**test 3: total ** (40% insert, 40% delete, 20% search) - everything's changing all the time lol

---

## what it prints out

when it runs:



then it does the same thing for the fancy locking, then shows which one won.

---

## how to run it

just do:
```bash
go run hw3_linked_lists.go
```

takes like 2-3 mins. that's it.

if you want the compiled version:
```bash
go build -o hw3.exe hw3_linked_lists.go
.\hw3.exe
```

---

## what the benchmark code does

inside main() there's basically:

1. make two fresh lists
2. put 100 nodes in each
3. start up a bunch of goroutines
4. each goroutine just does random insert/delete/search for 10k times
5. count how many actually finished
6. divide by how long it took
7. print the number

each thread just does:
```go
for i := 0; i < 10000; i++ {
    rand := pick a random number
    
    if random says insert {
        add something
    } else if random says delete {
        remove something
    } else {
        search for something
    }
}
```

yeah that's it. just punching the list over and over and seeing how fast it breaks.

---

## the results

**the simple way totally destroys the fancy way**
- simple lock: 1-4 million ops/sec
- fancy locks: 0.3-0.5 million ops/sec
- literal 5-10x faster

**why tho?**
our list is tiny (100 nodes). so with fancy locking you spend all your time grabbing/releasing locks on each node. you don't actually save any time.

all the fancy overhead kills it.

**but if the list was like 10,000 nodes?**
then yeah fancy would probably win because threads could work on completely different parts without waiting around.

---

## part 1 is in a different file

read `hw3_design_principle.md` for the theory stuff.

this file is just the code and how it works.

---

## just run it

```bash
go run hw3_linked_lists.go
```

watch it go. see which one wins. that's the whole thing.
````
