import json
import secrets

from Crypto.Hash import keccak

file_path = './mock.jsonl'

# Starting blockNumber
block_number = 0

# Maximum block number to reach
max_block_number = 100

def random_eth_hash():
    random_bytes = secrets.token_bytes(32)
    keccak_hash = keccak.new(digest_bits=256)
    keccak_hash.update(random_bytes)
    return keccak_hash.hexdigest()

def generate_big_integer():
    # return random.randint(1_000_000_000_000_000_000, 1_000_000_000_000_000_000_000)
    return 1


with open(file_path, 'w') as file:
    parent_hash = "0x0000000000000000000000000000000000000000000000000000000000000000"

    for block_number in range(1, max_block_number + 1):
        hash = '0x' + random_eth_hash()

        data = {
            "delta": generate_big_integer(),
            "reward": generate_big_integer(),
            "withdrawals": generate_big_integer(),
            "burn": generate_big_integer(),
            "blockNumber": block_number,
            "hash": hash,
            "parentHash": parent_hash
        }

        file.write(json.dumps(data) + '\n')

        # Update parent hash for the next block
        parent_hash = hash
