# Walrus 🦦
> Learning and trying to implement WAL from scratch.

This sequential storage provides:
1. __Easy Replay__
2. __High Write-Throughput__
3. __Data Integrity__

- Often perceived as simple append-only logs, especially when part of a larger sysem like Databases, RAFT, or LSM Trees, WALs are intricate structures essential for system reliability. Their functionality extends beyond basic logging.
- Each implementation of WAL is tailored to its specific environment, influencing overall system architecture far more than one might think. A WAL written for a consensus algorithm might be different from what database use.
- Although specific implementations and the exposed algorithms might change, certain key functionalities remain the same.
    1. __WriteEntry__: Allows adding new entries to the log.
    2. __ReadAll/Recover__: Enables the recovery of all entries from the log, or its most recent segment, ensuring that data can be restored in the event of system failure.
    3. __LogRotation__: To enhance efficiency during startup and recovery, the log is divided into multiple files. This approach also aids into managing log size.
    4. __Integrity Checks__: A vital aspect of WALs is ensuring the unaltered state of data on disk. Integrity checks confirm that the stored data has not been modified or corrupted, maintaining data reliability.
    5. __Auto-Repair/Recovery__: In cases of minor corruption, typically resulting from a crash, the log is designed to autonomously repair itself, preserving the continuity and integrity of the systems.

## Writes
A WAL must operate efficiently to prevent becoming a bottleneck for the main system it supports. Fast rewrites are crucial, and several factors must be considered to ensure this:
1. __Serialization Performance__: Data to be written to the disk must first be serialized into a byte stream. This process should be quick and efficient, producing a compact byte stream to minimize file size and write time. Many systems use Google's Protocol Buffers for this purpose, which provides rapid serialization and efficient encoding. Their strong type guarantees also simplify data reading and writing.
2. __Write Performance__: To optimize write times, the WAL should append data only at the end of the log, eliminating the need of disk seeks before writing. Sequential writes are not just faster in HDDs but also on SSDs (because of the way writes and garbage collection is handled in SSDs).
3. __In-Memory Buffers with Periodic Sync__: Rather than writing data directly to disk, WALs typically use in-memory buffers to store incoming writes, flushing these buffers to disk at set intervals. This flush frequency is adjustable and should be calibrated based on the system's needs. Shorter intervals lead to better durability but lead to slow writes, while longer intervals trade off durability for speed.
4. __Fsync__: Operating systems use their own buffers for disk file operations, which means data written to disk might initially be stored in an OS buffer and can be lost in case of power failure. This operating systems provide an FSync API to force the synchronization of the OS buffer with it's own disk.
5. __Checksums__: To ensure the integrity of data, each log entry should include a checksum. This checksum is vital during reads and repairs, helping to identify and eliminate corrupted entries. It adds a layer of data verification, further securing the log against errors.

## Reads
It's essential to understand that WALs are written much more often than read from. This high write-to-read ratio underscores the WAL's primary function in recording data changes for durability. Additionally, it's important to note that many WAL implementations do not support simultaneous reading and writing. This design choice is practical because reading from the WAL typically occurs only during the startup and recovery phases when the system is not actively processing new requests.

When reading, the WAL processes entries sequentially, starting from the beginning of the file and moving towards the end. An integral part of this process is verifying the checksums for each entry to ensure data integrity. If a checksum verification fails, the WAL halts the reading process and returns an error. This indicates that the WAL requires repair before it can be successfully read again. In cases where a single WAL entry is found to be corrupted, all subsequent entries must be discarded. This is because, beyond the point of corruption, there's no longer a reliable guarantee of data integrity, and continuing to process potentially compromised entries could lead to further inconsistencies or data loss.


## Functional Requirements
- write entries to a log
- read all entries from a given log segment
- read all entries from the last log segment
- read all entries from a given timestamp
- auto remove old log segments on reaching segment limit
- sync entries to disk at regular entries
- CRC32 checksum for data integrity
