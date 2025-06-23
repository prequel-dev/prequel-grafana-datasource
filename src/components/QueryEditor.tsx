import React, { ChangeEvent } from 'react';
import { InlineField, Input, Stack } from '@grafana/ui';
import { QueryEditorProps } from '@grafana/data';
import { DataSource } from '../datasource';
import { DataSourceOptions, PrequelDataQuery } from '../types';

type Props = QueryEditorProps<DataSource, PrequelDataQuery, DataSourceOptions>;

export function QueryEditor({ query, onChange, onRunQuery }: Props) {
  const onFilterChange = (event: ChangeEvent<HTMLInputElement>) => {
    onChange({ ...query, filter: event.target.value });
  };

  // const onConstantChange = (event: ChangeEvent<HTMLInputElement>) => {
  //   onChange({ ...query, constant: parseFloat(event.target.value) });
  //   // executes the query
  //   onRunQuery();
  // };

  const { filter } = query;

  return (
    <Stack gap={0}>
      <InlineField label="Filter" labelWidth={16} tooltip="Not used yet">
        <Input
          id="query-editor-filter"
          onChange={onFilterChange}
          value={filter || ''}
          required
          placeholder="Enter a filter expression"
        />
      </InlineField>
    </Stack>
  );
}
