ctxwat
------

I'm not sure how to use contexts in a channel/pipeline-based design where a single execution context may fan out into an unknown number of child execution contexts that share a deadline.

## V0

I'm making a (needlessly) pipeline application for printing numbers somewhat in the style of <https://blog.golang.org/pipelines>. We have one goroutine that reads off a channel of ints and for each int "n" it writes the string versions of [0, n) to an output channel. We have another goroutine that reads from the channel of strings and prints them. In `main` we set up those two workers, send in the input and wait for everyone to complete. Relatively straightforward.

## V1

Concerned with the performance of our string printing, we wanted to bound the execution of these using `context.Context`. For each int we're going to set a deadline of 100ms to both split that int into its messages _and_ print those messages. Each time we read an int we create a new context with a 100ms timeout and use that context to do our splitting operation. Then we pass the context along the channel to the executor goroutine which also uses the context to actually execute the message.

This code compiles and runs as before, however `go vet` complains (rightly) that in the successful execution case we fail to call the `CancelFunc` for our context, which leaks the context and any associated resources until it times out.

> The WithCancel, WithDeadline, and WithTimeout functions take a Context (the parent) and return a derived Context (the child) and a CancelFunc. Calling the CancelFunc cancels the child and its children, removes the parent's reference to the child, and stops any associated timers. Failing to call the CancelFunc leaks the child and its children until the parent is canceled or the timer fires. The go vet tool checks that CancelFuncs are used on all control-flow paths.

## V2

In an attempt to address this, I first looked at passing the `CancelFunc` along to the executor goroutine, but this runs into a problem pretty quickly. Since all the child "jobs" share a context, the first one will complete and cancel and then all the others will be cancelled. When I run this on my machine this causes only the first child (always "0") to get printed and the rest are skipped.

## V3

I tried to fix the problem above by creating child contexts with a separate `CancelFunc` from the parent. Now each child can cancel when it is complete, but `go vet` is back to complaining (correctly) that the parent context still leaks.

## V4

To ensure that the parent context is eventually cancelled when all the children are complete, I create a separate goroutine with a waitgroup that just watches for all the children to report that they are done. This ensures that the parent context will be cancelled when all the work is done, but this is a relatively complicated setup and it gets even more complicated if your "child" execution contexts can have children of their own. It also feels like it breaks the cleanliness of the basic design. Part of what makes the "pipelines" concept nice is that each step of the pipeline only needs to care about its own operation. Needing the step after you to report back to you when work was completed removes that cleanliness.

## V5

Since the real thing that we "care" about is that all our execution happens within a deadline, in v5 I tried to just initialize and pass around that deadline insted of a context created with that deadline. On the one hand this solves the problem of leaking context since each one is scoped to only a particular step of the pipeline. However this introduces a good amount of "busy work" for each step to initialize and use a `context.WithDeadline` for any execution that needs to be bounded. We also lose the ability to piggyback off of nice things `context.Context` gives us (propagating early cancellation, context scoped values, etc.) and we lose some affinity for the "normal" pattern of passing along a single `context.Context` through a thread of execution. We weren't using those things in this toy example, but they might be useful later.

## Questions?

None of these options seem perfect. It seems at a high level like I can:

* Pass one context throughout execution and add logic to "fan in" when work is done to ensure it does not leak.
* Create one context per step and pass around the overall deadline if that's whats important.
* Just leak the context knowing that it will time out eventually and tell `go vet` to ignore it.

Given that all of these seem to have problems, I have to ask:

* Am I thinking about or using `context.Context` incorrectly here?
* What is the cost of leaking contexts like this?
* Is there a cleaner way to thread through a context that might "fan out" without leaking or prematurely cancelling?
* Are there good examples of people solving this problem in the wild?
