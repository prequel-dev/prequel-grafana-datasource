import React, { ChangeEvent } from 'react';
import { InlineField, Input, SecretInput } from '@grafana/ui';
import { DataSourcePluginOptionsEditorProps } from '@grafana/data';
import { DataSourceOptions, SecureJsonData } from '../types';

interface Props extends DataSourcePluginOptionsEditorProps<DataSourceOptions, SecureJsonData> {}

export function ConfigEditor(props: Props) {
  const { onOptionsChange, options } = props;
  const { jsonData, secureJsonFields, secureJsonData } = options;

  const onURLChange = (event: ChangeEvent<HTMLInputElement>) => {
    onOptionsChange({
      ...options,
      jsonData: {
        ...jsonData,
        url: event.target.value,
      },
    });
  };

  // Secure field (only sent to the backend)
  const onTokenChange = (event: ChangeEvent<HTMLInputElement>) => {
    onOptionsChange({
      ...options,
      secureJsonData: {
        token: event.target.value,
      },
    });
  };

  const onResetToken = () => {
    onOptionsChange({
      ...options,
      secureJsonFields: {
        ...options.secureJsonFields,
        token: false,
      },
      secureJsonData: {
        ...options.secureJsonData,
        token: '',
      },
    });
  };

  return (
    <>
      <InlineField label="URL" labelWidth={14} interactive tooltip={'Json field returned to frontend'}>
        <Input
          id="config-editor-path"
          onChange={onURLChange}
          value={jsonData.url}
          placeholder="Enter URL, e.g. https://app-beta.prequel.dev"
          width={40}
        />
      </InlineField>
      <InlineField label="Token" labelWidth={14} interactive tooltip={'Secure json field (backend only)'}>
        <SecretInput
          required
          id="config-editor-api-key"
          isConfigured={secureJsonFields.token}
          value={secureJsonData?.token}
          placeholder="Enter API token"
          width={40}
          onReset={onResetToken}
          onChange={onTokenChange}
        />
      </InlineField>
    </>
  );
}
