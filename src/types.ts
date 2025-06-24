import { DataSourceJsonData } from '@grafana/data';
import { DataQuery } from '@grafana/schema';

export interface PreqQuery extends DataQuery {
  queryText?: string;
}

export const DEFAULT_QUERY: Partial<PreqQuery> = {
};

/**
 * These are options configured for each DataSource instance
 */
export interface PreqDataSourceOptions extends DataSourceJsonData {
  url?: string;
}

/**
 * Value that is used in the backend, but never sent over HTTP to the frontend
 */
export interface PreqSecureJsonData {
  token?: string;
}
