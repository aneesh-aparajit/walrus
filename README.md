# walrus
Implementation of Write-Ahead-Logs (WALs).

- __Functional Requirements__:
    - Write logs onto file
        - have explicit fsync file
            - might be required because of the CPU buffer, fsync generally is a slow operation.
    - Read from a specific log sequence number
    - Rotate logs
    - Repair Logs
        - in case some buffer is corrupted, truncate the file with just that the previously read buffers.
