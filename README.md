# Supply Tracer Parser

Supply Reader is a Go application that parses and sums supply data from a JSONL file. It's designed to handle large amounts of data efficiently and provides an API to expose the latest state of the parsed data.

## Features

- Reads supply data from a JSONL file, including support for reading log rotated files.
- Continues listening for new data when the file is updated.
- Stores the latest state for subsequent runs in a state file.
- Exposes the latest state through an API.
- Supports a "fresh" mode to start from scratch by removing the existing state file.

## Supply data provider

The supply JSONL file can be retrieved using:

```sh
geth --vmtrace supply --vmtrace.config '{"path": "."}'
```

## Usage

You can run the application with the following command:

```sh
go build
./supply-tracer-parser --supply.file your_supply_file.jsonl
```

> Replace `your_supply_file.jsonl` with the path to your supply data file.

## Flags

- `--supply.file`: The file to read supply data from. Supports reading log rotated files.
- `--state.file`: The file to store the latest state for subsequent runs.
- `--api.port`: The API port to expose the latest state.
- `--fresh`: Nuke the state and start fresh.

## API

The application exposes an API on the port specified by the `--api.port` flag. The API provides the latest state of the parsed supply data.

## Mock Data

You can generate mock data using the provided Python script `mock_generator.py`. This script generates a JSONL file with mock supply data.

## Testing

You can run tests with the following command:

```sh
go test ./...
```

## Contributing

Contributions are welcome. Please submit a pull request or create an issue to discuss the changes you want to make.

## License

This project is licensed under the MIT License.