#!/usr/bin/env python3
"""
Wrapper that converts all outputs (DataFrame and RDF Graph) to JSON format for Go
"""
import sys
import json
import os
import tempfile
import logging
import time
from datetime import datetime
import pandas as pd
from rdflib import Graph

# Setup
script_dir = os.path.dirname(os.path.abspath(__file__))
os.chdir(script_dir)

logger = logging.getLogger('Symbolic_predictions')
logger.setLevel(logging.INFO)

logs_dir = 'logs'
if not os.path.exists(logs_dir):
    os.makedirs(logs_dir)

timestamp = time.strftime('%Y%m%d-%H%M%S')
log_file = os.path.join(logs_dir, f'full_data_{timestamp}.log')

# Log to stderr so it doesn't interfere with JSON output
handler = logging.StreamHandler(sys.stderr)
handler.setFormatter(logging.Formatter('%(asctime)s - %(levelname)s - %(message)s'))
logger.addHandler(handler)

import Symbolic_predictions

Symbolic_predictions.logger = logger


def dataframe_to_json(df):
    """Convert pandas DataFrame to JSON-serializable format"""
    if df is None or df.empty:
        return {
            "columns": [],
            "data": [],
            "shape": [0, 0]
        }

    return {
        "columns": df.columns.tolist(),
        "data": df.to_dict('records'),  # List of dicts
        "shape": list(df.shape),
        "dtypes": df.dtypes.astype(str).to_dict()
    }


def graph_to_json(graph, limit=None):
    """Convert RDF Graph to JSON-serializable format"""
    triples = []
    namespaces = {}

    # Get namespaces
    for prefix, uri in graph.namespaces():
        namespaces[prefix] = str(uri)

    # Get triples
    count = 0
    for s, p, o in graph:
        triples.append({
            "subject": str(s),
            "predicate": str(p),
            "object": str(o),
            "object_type": "literal" if hasattr(o, 'datatype') else "uri"
        })
        count += 1
        if limit and count >= limit:
            break

    return {
        "triples": triples,
        "total_triples": len(graph),
        "namespaces": namespaces,
        "limited_to": limit if limit else None
    }


def capture_sparql_queries():
    """Capture SPARQL queries during execution"""
    queries = []

    original_load_graph = Symbolic_predictions.load_graph

    def patched_load_graph(file):
        g = original_load_graph(file)

        # Wrap the query method
        original_query = g.query

        def capturing_query(query_object):
            query_str = str(query_object)
            start_time = time.time()

            # Execute query
            results = original_query(query_object)

            # Convert results to list to count them
            results_list = list(results)

            queries.append({
                "query": query_str,
                "execution_time": time.time() - start_time,
                "result_count": len(results_list),
                "timestamp": datetime.now().isoformat()
            })

            # Re-create results since we consumed them
            return original_query(query_object)

        g.query = capturing_query
        return g

    Symbolic_predictions.load_graph = patched_load_graph
    return queries


def main():
    # Setup query capture
    captured_queries = capture_sparql_queries()

    # Read input
    input_data = json.load(sys.stdin)

    # Create temporary config file
    with tempfile.NamedTemporaryFile(mode='w', suffix='.json', delete=False) as f:
        json.dump(input_data, f)
        temp_config_path = f.name

    try:
        start_time = time.time()

        # Initialize
        prefix, rulesfile, rdf_data, path, predictions_folder, constraints, kg, pca_threshold = Symbolic_predictions.initialize(
            temp_config_path)

        # Load initial graph for comparison
        initial_graph = Graph()
        initial_graph.parse(rdf_data, format='nt')
        initial_graph_json = graph_to_json(initial_graph, limit=100)  # Limit for performance

        # Process rules - this is the main function that generates predictions
        result_df, enriched_kg = Symbolic_predictions.process_rules(
            rulesfile, prefix, rdf_data, predictions_folder, kg, pca_threshold
        )

        # Convert results to JSON
        predictions_json = dataframe_to_json(result_df)

        # Convert enriched graph to JSON (limit to avoid huge outputs)
        enriched_graph_json = graph_to_json(enriched_kg, limit=1000)

        # Get new triples only (predictions)
        new_triples = []
        if result_df is not None and not result_df.empty:
            for _, row in result_df.iterrows():
                new_triples.append({
                    "subject": row['subject'],
                    "predicate": row['predicate'],
                    "object": row['object']
                })

        # Prepare complete output with all data
        output = {
            "success": True,
            "execution_time": time.time() - start_time,
            "timestamp": datetime.now().isoformat(),

            # Complete predictions DataFrame as JSON
            "predictions_dataframe": predictions_json,

            # New triples (predictions only)
            "new_triples": new_triples,

            # Graph data
            "graphs": {
                "initial": initial_graph_json,
                "enriched": enriched_graph_json,
                "statistics": {
                    "initial_triples": initial_graph_json["total_triples"],
                    "enriched_triples": enriched_graph_json["total_triples"],
                    "predictions_added": len(new_triples)
                }
            },

            # SPARQL queries executed
            "sparql_queries": captured_queries,

            # File outputs
            "output_files": {
                "predictions_folder": predictions_folder,
                "enriched_kg_path": os.path.join(os.path.dirname(predictions_folder), f"{kg}_EnrichedKG",
                                                 f"{kg}_Enriched_KG.nt"),
                "log_file": log_file
            },

            # Summary
            "summary": {
                "total_predictions": len(new_triples),
                "queries_executed": len(captured_queries),
                "processing_successful": True
            }
        }

        print(json.dumps(output, indent=2))

    except Exception as e:
        import traceback
        error_output = {
            "success": False,
            "error": str(e),
            "traceback": traceback.format_exc(),
            "timestamp": datetime.now().isoformat()
        }
        print(json.dumps(error_output, indent=2))
        sys.exit(1)

    finally:
        if os.path.exists(temp_config_path):
            os.remove(temp_config_path)


if __name__ == '__main__':
    main()