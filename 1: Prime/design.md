# 1: Prime Time

## Overview
1. Start up multiple worker threads, ready to handle incoming connections
2. Listen for connections
3. Handle incoming requests, responding correctly
    a. Discern whether value is a float, int, or invalid
    b. Handle big numbers
    c. Cacluate whether a value is prime

## Challenges
1. Handle large numbers
2. Handle floats
 
## Data Flow
- Data comes in from connection 
- Read all data from the connection
- Separate the data by newline character
- Parse the data into request