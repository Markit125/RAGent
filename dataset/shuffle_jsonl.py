import random
import sys

def clean_and_shuffle(input_file, output_file):
    """
    Reads a JSONL file, removes duplicate lines, shuffles them, 
    and writes to a new file.
    """
    try:
        print(f"Reading from {input_file}...")
        
        with open(input_file, 'r', encoding='utf-8') as f:
            lines = f.readlines()

        if not lines:
            print("Error: The input file is empty.")
            return

        original_count = len(lines)
        
        # 1. Deduplicate using a set (removes exact text matches)
        # We strip whitespace to ensure lines with different indentation/newlines are caught
        unique_lines_set = {line.strip() for line in lines if line.strip()}
        
        # Convert back to list for shuffling
        # We add the newline character back to maintain JSONL format
        processed_lines = [line + '\n' for line in unique_lines_set]
        
        final_count = len(processed_lines)
        duplicates_removed = original_count - final_count

        print(f"Original lines: {original_count}")
        print(f"Duplicates removed: {duplicates_removed}")
        print(f"Unique lines: {final_count}")

        # 2. Shuffle
        print("Shuffling data...")
        random.shuffle(processed_lines)

        # 3. Write to output
        with open(output_file, 'w', encoding='utf-8') as f:
            f.writelines(processed_lines)

        print(f"Success! Cleaned data written to {output_file}")

    except FileNotFoundError:
        print(f"Error: The file '{input_file}' was not found.")
    except Exception as e:
        print(f"An error occurred: {e}")

if __name__ == "__main__":
    # Configuration
    INPUT_FILENAME = 'dataset_shuffled.jsonl'
    OUTPUT_FILENAME = 'dataset_clean.jsonl'

    clean_and_shuffle(INPUT_FILENAME, OUTPUT_FILENAME)