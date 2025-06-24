import { DataSourceInstanceSettings, CoreApp, ScopedVars } from '@grafana/data';
import { DataSourceWithBackend, getTemplateSrv } from '@grafana/runtime';

import { PreqQuery, PreqDataSourceOptions, DEFAULT_QUERY } from './types';

export class DataSource extends DataSourceWithBackend<PreqQuery, PreqDataSourceOptions> {
  constructor(instanceSettings: DataSourceInstanceSettings<PreqDataSourceOptions>) {
    super(instanceSettings);
    this.annotations = {
      getDefaultQuery() {
        return {
          ...DEFAULT_QUERY,
          queryType: 'annotations',
        };
      },
    };
  }

  getDefaultQuery(_: CoreApp): Partial<PreqQuery> {
    return DEFAULT_QUERY;
  }

  applyTemplateVariables(query: PreqQuery, scopedVars: ScopedVars) {
    return {
      ...query,
      queryText: getTemplateSrv().replace(query.queryText, scopedVars),
    };
  }

  filterQuery(query: PreqQuery): boolean {
    // if no query has been provided, prevent the query from being executed
    // return !!query.queryText;
    return true;
  }
}
