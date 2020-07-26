# Example

## Run
First, make sure the kpt function is available in the parent directory. It 
can be built with the following command:
```
(cd ..; make build)
```

Execute the following command from the current directory:
```
kpt fn source my-pkg | kpt fn run --enable-exec --exec-path ./../localpackage-fn | kpt fn sink
```
This prints the output to stdout.