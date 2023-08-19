export const popular_findMany = {
  "fieldName": "populars",
  "fieldNodes": [
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "populars",
        "loc": {
          "start": 7949,
          "end": 7957
        }
      },
      "arguments": [
        {
          "kind": "Argument",
          "name": {
            "kind": "Name",
            "value": "input",
            "loc": {
              "start": 7958,
              "end": 7963
            }
          },
          "value": {
            "kind": "Variable",
            "name": {
              "kind": "Name",
              "value": "input",
              "loc": {
                "start": 7966,
                "end": 7971
              }
            },
            "loc": {
              "start": 7965,
              "end": 7971
            }
          },
          "loc": {
            "start": 7958,
            "end": 7971
          }
        }
      ],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "edges",
              "loc": {
                "start": 7979,
                "end": 7984
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "cursor",
                    "loc": {
                      "start": 7995,
                      "end": 8001
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7995,
                    "end": 8001
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "node",
                    "loc": {
                      "start": 8010,
                      "end": 8014
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "InlineFragment",
                        "typeCondition": {
                          "kind": "NamedType",
                          "name": {
                            "kind": "Name",
                            "value": "Api",
                            "loc": {
                              "start": 8036,
                              "end": 8039
                            }
                          },
                          "loc": {
                            "start": 8036,
                            "end": 8039
                          }
                        },
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "FragmentSpread",
                              "name": {
                                "kind": "Name",
                                "value": "Api_list",
                                "loc": {
                                  "start": 8061,
                                  "end": 8069
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 8058,
                                "end": 8069
                              }
                            }
                          ],
                          "loc": {
                            "start": 8040,
                            "end": 8083
                          }
                        },
                        "loc": {
                          "start": 8029,
                          "end": 8083
                        }
                      },
                      {
                        "kind": "InlineFragment",
                        "typeCondition": {
                          "kind": "NamedType",
                          "name": {
                            "kind": "Name",
                            "value": "Note",
                            "loc": {
                              "start": 8103,
                              "end": 8107
                            }
                          },
                          "loc": {
                            "start": 8103,
                            "end": 8107
                          }
                        },
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "FragmentSpread",
                              "name": {
                                "kind": "Name",
                                "value": "Note_list",
                                "loc": {
                                  "start": 8129,
                                  "end": 8138
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 8126,
                                "end": 8138
                              }
                            }
                          ],
                          "loc": {
                            "start": 8108,
                            "end": 8152
                          }
                        },
                        "loc": {
                          "start": 8096,
                          "end": 8152
                        }
                      },
                      {
                        "kind": "InlineFragment",
                        "typeCondition": {
                          "kind": "NamedType",
                          "name": {
                            "kind": "Name",
                            "value": "Organization",
                            "loc": {
                              "start": 8172,
                              "end": 8184
                            }
                          },
                          "loc": {
                            "start": 8172,
                            "end": 8184
                          }
                        },
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "FragmentSpread",
                              "name": {
                                "kind": "Name",
                                "value": "Organization_list",
                                "loc": {
                                  "start": 8206,
                                  "end": 8223
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 8203,
                                "end": 8223
                              }
                            }
                          ],
                          "loc": {
                            "start": 8185,
                            "end": 8237
                          }
                        },
                        "loc": {
                          "start": 8165,
                          "end": 8237
                        }
                      },
                      {
                        "kind": "InlineFragment",
                        "typeCondition": {
                          "kind": "NamedType",
                          "name": {
                            "kind": "Name",
                            "value": "Project",
                            "loc": {
                              "start": 8257,
                              "end": 8264
                            }
                          },
                          "loc": {
                            "start": 8257,
                            "end": 8264
                          }
                        },
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "FragmentSpread",
                              "name": {
                                "kind": "Name",
                                "value": "Project_list",
                                "loc": {
                                  "start": 8286,
                                  "end": 8298
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 8283,
                                "end": 8298
                              }
                            }
                          ],
                          "loc": {
                            "start": 8265,
                            "end": 8312
                          }
                        },
                        "loc": {
                          "start": 8250,
                          "end": 8312
                        }
                      },
                      {
                        "kind": "InlineFragment",
                        "typeCondition": {
                          "kind": "NamedType",
                          "name": {
                            "kind": "Name",
                            "value": "Question",
                            "loc": {
                              "start": 8332,
                              "end": 8340
                            }
                          },
                          "loc": {
                            "start": 8332,
                            "end": 8340
                          }
                        },
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "FragmentSpread",
                              "name": {
                                "kind": "Name",
                                "value": "Question_list",
                                "loc": {
                                  "start": 8362,
                                  "end": 8375
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 8359,
                                "end": 8375
                              }
                            }
                          ],
                          "loc": {
                            "start": 8341,
                            "end": 8389
                          }
                        },
                        "loc": {
                          "start": 8325,
                          "end": 8389
                        }
                      },
                      {
                        "kind": "InlineFragment",
                        "typeCondition": {
                          "kind": "NamedType",
                          "name": {
                            "kind": "Name",
                            "value": "Routine",
                            "loc": {
                              "start": 8409,
                              "end": 8416
                            }
                          },
                          "loc": {
                            "start": 8409,
                            "end": 8416
                          }
                        },
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "FragmentSpread",
                              "name": {
                                "kind": "Name",
                                "value": "Routine_list",
                                "loc": {
                                  "start": 8438,
                                  "end": 8450
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 8435,
                                "end": 8450
                              }
                            }
                          ],
                          "loc": {
                            "start": 8417,
                            "end": 8464
                          }
                        },
                        "loc": {
                          "start": 8402,
                          "end": 8464
                        }
                      },
                      {
                        "kind": "InlineFragment",
                        "typeCondition": {
                          "kind": "NamedType",
                          "name": {
                            "kind": "Name",
                            "value": "SmartContract",
                            "loc": {
                              "start": 8484,
                              "end": 8497
                            }
                          },
                          "loc": {
                            "start": 8484,
                            "end": 8497
                          }
                        },
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "FragmentSpread",
                              "name": {
                                "kind": "Name",
                                "value": "SmartContract_list",
                                "loc": {
                                  "start": 8519,
                                  "end": 8537
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 8516,
                                "end": 8537
                              }
                            }
                          ],
                          "loc": {
                            "start": 8498,
                            "end": 8551
                          }
                        },
                        "loc": {
                          "start": 8477,
                          "end": 8551
                        }
                      },
                      {
                        "kind": "InlineFragment",
                        "typeCondition": {
                          "kind": "NamedType",
                          "name": {
                            "kind": "Name",
                            "value": "Standard",
                            "loc": {
                              "start": 8571,
                              "end": 8579
                            }
                          },
                          "loc": {
                            "start": 8571,
                            "end": 8579
                          }
                        },
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "FragmentSpread",
                              "name": {
                                "kind": "Name",
                                "value": "Standard_list",
                                "loc": {
                                  "start": 8601,
                                  "end": 8614
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 8598,
                                "end": 8614
                              }
                            }
                          ],
                          "loc": {
                            "start": 8580,
                            "end": 8628
                          }
                        },
                        "loc": {
                          "start": 8564,
                          "end": 8628
                        }
                      },
                      {
                        "kind": "InlineFragment",
                        "typeCondition": {
                          "kind": "NamedType",
                          "name": {
                            "kind": "Name",
                            "value": "User",
                            "loc": {
                              "start": 8648,
                              "end": 8652
                            }
                          },
                          "loc": {
                            "start": 8648,
                            "end": 8652
                          }
                        },
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "FragmentSpread",
                              "name": {
                                "kind": "Name",
                                "value": "User_list",
                                "loc": {
                                  "start": 8674,
                                  "end": 8683
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 8671,
                                "end": 8683
                              }
                            }
                          ],
                          "loc": {
                            "start": 8653,
                            "end": 8697
                          }
                        },
                        "loc": {
                          "start": 8641,
                          "end": 8697
                        }
                      }
                    ],
                    "loc": {
                      "start": 8015,
                      "end": 8707
                    }
                  },
                  "loc": {
                    "start": 8010,
                    "end": 8707
                  }
                }
              ],
              "loc": {
                "start": 7985,
                "end": 8713
              }
            },
            "loc": {
              "start": 7979,
              "end": 8713
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "pageInfo",
              "loc": {
                "start": 8718,
                "end": 8726
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "hasNextPage",
                    "loc": {
                      "start": 8737,
                      "end": 8748
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8737,
                    "end": 8748
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorApi",
                    "loc": {
                      "start": 8757,
                      "end": 8769
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8757,
                    "end": 8769
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorNote",
                    "loc": {
                      "start": 8778,
                      "end": 8791
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8778,
                    "end": 8791
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorOrganization",
                    "loc": {
                      "start": 8800,
                      "end": 8821
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8800,
                    "end": 8821
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorProject",
                    "loc": {
                      "start": 8830,
                      "end": 8846
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8830,
                    "end": 8846
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorQuestion",
                    "loc": {
                      "start": 8855,
                      "end": 8872
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8855,
                    "end": 8872
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorRoutine",
                    "loc": {
                      "start": 8881,
                      "end": 8897
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8881,
                    "end": 8897
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorSmartContract",
                    "loc": {
                      "start": 8906,
                      "end": 8928
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8906,
                    "end": 8928
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorStandard",
                    "loc": {
                      "start": 8937,
                      "end": 8954
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8937,
                    "end": 8954
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorUser",
                    "loc": {
                      "start": 8963,
                      "end": 8976
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8963,
                    "end": 8976
                  }
                }
              ],
              "loc": {
                "start": 8727,
                "end": 8982
              }
            },
            "loc": {
              "start": 8718,
              "end": 8982
            }
          }
        ],
        "loc": {
          "start": 7973,
          "end": 8986
        }
      },
      "loc": {
        "start": 7949,
        "end": 8986
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "versions",
        "loc": {
          "start": 28,
          "end": 36
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 43,
                "end": 55
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 66,
                      "end": 68
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 66,
                    "end": 68
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 77,
                      "end": 85
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 77,
                    "end": 85
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "details",
                    "loc": {
                      "start": 94,
                      "end": 101
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 94,
                    "end": 101
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 110,
                      "end": 114
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 110,
                    "end": 114
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "summary",
                    "loc": {
                      "start": 123,
                      "end": 130
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 123,
                    "end": 130
                  }
                }
              ],
              "loc": {
                "start": 56,
                "end": 136
              }
            },
            "loc": {
              "start": 43,
              "end": 136
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 141,
                "end": 143
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 141,
              "end": 143
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 148,
                "end": 158
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 148,
              "end": 158
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 163,
                "end": 173
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 163,
              "end": 173
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "callLink",
              "loc": {
                "start": 178,
                "end": 186
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 178,
              "end": 186
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "commentsCount",
              "loc": {
                "start": 191,
                "end": 204
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 191,
              "end": 204
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "documentationLink",
              "loc": {
                "start": 209,
                "end": 226
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 209,
              "end": 226
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "forksCount",
              "loc": {
                "start": 231,
                "end": 241
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 231,
              "end": 241
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLatest",
              "loc": {
                "start": 246,
                "end": 254
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 246,
              "end": 254
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 259,
                "end": 268
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 259,
              "end": 268
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 273,
                "end": 285
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 273,
              "end": 285
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionIndex",
              "loc": {
                "start": 290,
                "end": 302
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 290,
              "end": 302
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionLabel",
              "loc": {
                "start": 307,
                "end": 319
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 307,
              "end": 319
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 324,
                "end": 327
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canComment",
                    "loc": {
                      "start": 338,
                      "end": 348
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 338,
                    "end": 348
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canCopy",
                    "loc": {
                      "start": 357,
                      "end": 364
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 357,
                    "end": 364
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 373,
                      "end": 382
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 373,
                    "end": 382
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 391,
                      "end": 400
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 391,
                    "end": 400
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 409,
                      "end": 418
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 409,
                    "end": 418
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUse",
                    "loc": {
                      "start": 427,
                      "end": 433
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 427,
                    "end": 433
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 442,
                      "end": 449
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 442,
                    "end": 449
                  }
                }
              ],
              "loc": {
                "start": 328,
                "end": 455
              }
            },
            "loc": {
              "start": 324,
              "end": 455
            }
          }
        ],
        "loc": {
          "start": 37,
          "end": 457
        }
      },
      "loc": {
        "start": 28,
        "end": 457
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 458,
          "end": 460
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 458,
        "end": 460
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 461,
          "end": 471
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 461,
        "end": 471
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 472,
          "end": 482
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 472,
        "end": 482
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 483,
          "end": 492
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 483,
        "end": 492
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "issuesCount",
        "loc": {
          "start": 493,
          "end": 504
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 493,
        "end": 504
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 505,
          "end": 511
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Label_list",
              "loc": {
                "start": 521,
                "end": 531
              }
            },
            "directives": [],
            "loc": {
              "start": 518,
              "end": 531
            }
          }
        ],
        "loc": {
          "start": 512,
          "end": 533
        }
      },
      "loc": {
        "start": 505,
        "end": 533
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 534,
          "end": 539
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Organization",
                "loc": {
                  "start": 553,
                  "end": 565
                }
              },
              "loc": {
                "start": 553,
                "end": 565
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Organization_nav",
                    "loc": {
                      "start": 579,
                      "end": 595
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 576,
                    "end": 595
                  }
                }
              ],
              "loc": {
                "start": 566,
                "end": 601
              }
            },
            "loc": {
              "start": 546,
              "end": 601
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "User",
                "loc": {
                  "start": 613,
                  "end": 617
                }
              },
              "loc": {
                "start": 613,
                "end": 617
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "User_nav",
                    "loc": {
                      "start": 631,
                      "end": 639
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 628,
                    "end": 639
                  }
                }
              ],
              "loc": {
                "start": 618,
                "end": 645
              }
            },
            "loc": {
              "start": 606,
              "end": 645
            }
          }
        ],
        "loc": {
          "start": 540,
          "end": 647
        }
      },
      "loc": {
        "start": 534,
        "end": 647
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "permissions",
        "loc": {
          "start": 648,
          "end": 659
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 648,
        "end": 659
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "questionsCount",
        "loc": {
          "start": 660,
          "end": 674
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 660,
        "end": 674
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 675,
          "end": 680
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 675,
        "end": 680
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 681,
          "end": 690
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 681,
        "end": 690
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 691,
          "end": 695
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Tag_list",
              "loc": {
                "start": 705,
                "end": 713
              }
            },
            "directives": [],
            "loc": {
              "start": 702,
              "end": 713
            }
          }
        ],
        "loc": {
          "start": 696,
          "end": 715
        }
      },
      "loc": {
        "start": 691,
        "end": 715
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "transfersCount",
        "loc": {
          "start": 716,
          "end": 730
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 716,
        "end": 730
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "views",
        "loc": {
          "start": 731,
          "end": 736
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 731,
        "end": 736
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 737,
          "end": 740
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 747,
                "end": 756
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 747,
              "end": 756
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 761,
                "end": 772
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 761,
              "end": 772
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canTransfer",
              "loc": {
                "start": 777,
                "end": 788
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 777,
              "end": 788
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 793,
                "end": 802
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 793,
              "end": 802
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 807,
                "end": 814
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 807,
              "end": 814
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReact",
              "loc": {
                "start": 819,
                "end": 827
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 819,
              "end": 827
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 832,
                "end": 844
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 832,
              "end": 844
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 849,
                "end": 857
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 849,
              "end": 857
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 862,
                "end": 870
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 862,
              "end": 870
            }
          }
        ],
        "loc": {
          "start": 741,
          "end": 872
        }
      },
      "loc": {
        "start": 737,
        "end": 872
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 901,
          "end": 903
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 901,
        "end": 903
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 904,
          "end": 913
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 904,
        "end": 913
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 947,
          "end": 949
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 947,
        "end": 949
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 950,
          "end": 960
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 950,
        "end": 960
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 961,
          "end": 971
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 961,
        "end": 971
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "color",
        "loc": {
          "start": 972,
          "end": 977
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 972,
        "end": 977
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "label",
        "loc": {
          "start": 978,
          "end": 983
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 978,
        "end": 983
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 984,
          "end": 989
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Organization",
                "loc": {
                  "start": 1003,
                  "end": 1015
                }
              },
              "loc": {
                "start": 1003,
                "end": 1015
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Organization_nav",
                    "loc": {
                      "start": 1029,
                      "end": 1045
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1026,
                    "end": 1045
                  }
                }
              ],
              "loc": {
                "start": 1016,
                "end": 1051
              }
            },
            "loc": {
              "start": 996,
              "end": 1051
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "User",
                "loc": {
                  "start": 1063,
                  "end": 1067
                }
              },
              "loc": {
                "start": 1063,
                "end": 1067
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "User_nav",
                    "loc": {
                      "start": 1081,
                      "end": 1089
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1078,
                    "end": 1089
                  }
                }
              ],
              "loc": {
                "start": 1068,
                "end": 1095
              }
            },
            "loc": {
              "start": 1056,
              "end": 1095
            }
          }
        ],
        "loc": {
          "start": 990,
          "end": 1097
        }
      },
      "loc": {
        "start": 984,
        "end": 1097
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 1098,
          "end": 1101
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 1108,
                "end": 1117
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1108,
              "end": 1117
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 1122,
                "end": 1131
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1122,
              "end": 1131
            }
          }
        ],
        "loc": {
          "start": 1102,
          "end": 1133
        }
      },
      "loc": {
        "start": 1098,
        "end": 1133
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "versions",
        "loc": {
          "start": 1165,
          "end": 1173
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 1180,
                "end": 1192
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 1203,
                      "end": 1205
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1203,
                    "end": 1205
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 1214,
                      "end": 1222
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1214,
                    "end": 1222
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 1231,
                      "end": 1242
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1231,
                    "end": 1242
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 1251,
                      "end": 1255
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1251,
                    "end": 1255
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "pages",
                    "loc": {
                      "start": 1264,
                      "end": 1269
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 1284,
                            "end": 1286
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1284,
                          "end": 1286
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "pageIndex",
                          "loc": {
                            "start": 1299,
                            "end": 1308
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1299,
                          "end": 1308
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "text",
                          "loc": {
                            "start": 1321,
                            "end": 1325
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1321,
                          "end": 1325
                        }
                      }
                    ],
                    "loc": {
                      "start": 1270,
                      "end": 1335
                    }
                  },
                  "loc": {
                    "start": 1264,
                    "end": 1335
                  }
                }
              ],
              "loc": {
                "start": 1193,
                "end": 1341
              }
            },
            "loc": {
              "start": 1180,
              "end": 1341
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 1346,
                "end": 1348
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1346,
              "end": 1348
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 1353,
                "end": 1363
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1353,
              "end": 1363
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 1368,
                "end": 1378
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1368,
              "end": 1378
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLatest",
              "loc": {
                "start": 1383,
                "end": 1391
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1383,
              "end": 1391
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 1396,
                "end": 1405
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1396,
              "end": 1405
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 1410,
                "end": 1422
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1410,
              "end": 1422
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionIndex",
              "loc": {
                "start": 1427,
                "end": 1439
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1427,
              "end": 1439
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionLabel",
              "loc": {
                "start": 1444,
                "end": 1456
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1444,
              "end": 1456
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 1461,
                "end": 1464
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canComment",
                    "loc": {
                      "start": 1475,
                      "end": 1485
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1475,
                    "end": 1485
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canCopy",
                    "loc": {
                      "start": 1494,
                      "end": 1501
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1494,
                    "end": 1501
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 1510,
                      "end": 1519
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1510,
                    "end": 1519
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 1528,
                      "end": 1537
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1528,
                    "end": 1537
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 1546,
                      "end": 1555
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1546,
                    "end": 1555
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUse",
                    "loc": {
                      "start": 1564,
                      "end": 1570
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1564,
                    "end": 1570
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 1579,
                      "end": 1586
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1579,
                    "end": 1586
                  }
                }
              ],
              "loc": {
                "start": 1465,
                "end": 1592
              }
            },
            "loc": {
              "start": 1461,
              "end": 1592
            }
          }
        ],
        "loc": {
          "start": 1174,
          "end": 1594
        }
      },
      "loc": {
        "start": 1165,
        "end": 1594
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 1595,
          "end": 1597
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1595,
        "end": 1597
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 1598,
          "end": 1608
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1598,
        "end": 1608
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 1609,
          "end": 1619
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1609,
        "end": 1619
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 1620,
          "end": 1629
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1620,
        "end": 1629
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "issuesCount",
        "loc": {
          "start": 1630,
          "end": 1641
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1630,
        "end": 1641
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 1642,
          "end": 1648
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Label_list",
              "loc": {
                "start": 1658,
                "end": 1668
              }
            },
            "directives": [],
            "loc": {
              "start": 1655,
              "end": 1668
            }
          }
        ],
        "loc": {
          "start": 1649,
          "end": 1670
        }
      },
      "loc": {
        "start": 1642,
        "end": 1670
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 1671,
          "end": 1676
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Organization",
                "loc": {
                  "start": 1690,
                  "end": 1702
                }
              },
              "loc": {
                "start": 1690,
                "end": 1702
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Organization_nav",
                    "loc": {
                      "start": 1716,
                      "end": 1732
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1713,
                    "end": 1732
                  }
                }
              ],
              "loc": {
                "start": 1703,
                "end": 1738
              }
            },
            "loc": {
              "start": 1683,
              "end": 1738
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "User",
                "loc": {
                  "start": 1750,
                  "end": 1754
                }
              },
              "loc": {
                "start": 1750,
                "end": 1754
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "User_nav",
                    "loc": {
                      "start": 1768,
                      "end": 1776
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1765,
                    "end": 1776
                  }
                }
              ],
              "loc": {
                "start": 1755,
                "end": 1782
              }
            },
            "loc": {
              "start": 1743,
              "end": 1782
            }
          }
        ],
        "loc": {
          "start": 1677,
          "end": 1784
        }
      },
      "loc": {
        "start": 1671,
        "end": 1784
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "permissions",
        "loc": {
          "start": 1785,
          "end": 1796
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1785,
        "end": 1796
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "questionsCount",
        "loc": {
          "start": 1797,
          "end": 1811
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1797,
        "end": 1811
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 1812,
          "end": 1817
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1812,
        "end": 1817
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 1818,
          "end": 1827
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1818,
        "end": 1827
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 1828,
          "end": 1832
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Tag_list",
              "loc": {
                "start": 1842,
                "end": 1850
              }
            },
            "directives": [],
            "loc": {
              "start": 1839,
              "end": 1850
            }
          }
        ],
        "loc": {
          "start": 1833,
          "end": 1852
        }
      },
      "loc": {
        "start": 1828,
        "end": 1852
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "transfersCount",
        "loc": {
          "start": 1853,
          "end": 1867
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1853,
        "end": 1867
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "views",
        "loc": {
          "start": 1868,
          "end": 1873
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1868,
        "end": 1873
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 1874,
          "end": 1877
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 1884,
                "end": 1893
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1884,
              "end": 1893
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 1898,
                "end": 1909
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1898,
              "end": 1909
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canTransfer",
              "loc": {
                "start": 1914,
                "end": 1925
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1914,
              "end": 1925
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 1930,
                "end": 1939
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1930,
              "end": 1939
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 1944,
                "end": 1951
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1944,
              "end": 1951
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReact",
              "loc": {
                "start": 1956,
                "end": 1964
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1956,
              "end": 1964
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 1969,
                "end": 1981
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1969,
              "end": 1981
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 1986,
                "end": 1994
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1986,
              "end": 1994
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 1999,
                "end": 2007
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1999,
              "end": 2007
            }
          }
        ],
        "loc": {
          "start": 1878,
          "end": 2009
        }
      },
      "loc": {
        "start": 1874,
        "end": 2009
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 2040,
          "end": 2042
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2040,
        "end": 2042
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 2043,
          "end": 2052
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2043,
        "end": 2052
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 2100,
          "end": 2102
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2100,
        "end": 2102
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bannerImage",
        "loc": {
          "start": 2103,
          "end": 2114
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2103,
        "end": 2114
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 2115,
          "end": 2121
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2115,
        "end": 2121
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 2122,
          "end": 2132
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2122,
        "end": 2132
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 2133,
          "end": 2143
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2133,
        "end": 2143
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isOpenToNewMembers",
        "loc": {
          "start": 2144,
          "end": 2162
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2144,
        "end": 2162
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 2163,
          "end": 2172
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2163,
        "end": 2172
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "commentsCount",
        "loc": {
          "start": 2173,
          "end": 2186
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2173,
        "end": 2186
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "membersCount",
        "loc": {
          "start": 2187,
          "end": 2199
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2187,
        "end": 2199
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "profileImage",
        "loc": {
          "start": 2200,
          "end": 2212
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2200,
        "end": 2212
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "reportsCount",
        "loc": {
          "start": 2213,
          "end": 2225
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2213,
        "end": 2225
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 2226,
          "end": 2235
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2226,
        "end": 2235
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 2236,
          "end": 2240
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Tag_list",
              "loc": {
                "start": 2250,
                "end": 2258
              }
            },
            "directives": [],
            "loc": {
              "start": 2247,
              "end": 2258
            }
          }
        ],
        "loc": {
          "start": 2241,
          "end": 2260
        }
      },
      "loc": {
        "start": 2236,
        "end": 2260
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "translations",
        "loc": {
          "start": 2261,
          "end": 2273
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 2280,
                "end": 2282
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2280,
              "end": 2282
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "language",
              "loc": {
                "start": 2287,
                "end": 2295
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2287,
              "end": 2295
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bio",
              "loc": {
                "start": 2300,
                "end": 2303
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2300,
              "end": 2303
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 2308,
                "end": 2312
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2308,
              "end": 2312
            }
          }
        ],
        "loc": {
          "start": 2274,
          "end": 2314
        }
      },
      "loc": {
        "start": 2261,
        "end": 2314
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 2315,
          "end": 2318
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canAddMembers",
              "loc": {
                "start": 2325,
                "end": 2338
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2325,
              "end": 2338
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 2343,
                "end": 2352
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2343,
              "end": 2352
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 2357,
                "end": 2368
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2357,
              "end": 2368
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReport",
              "loc": {
                "start": 2373,
                "end": 2382
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2373,
              "end": 2382
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 2387,
                "end": 2396
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2387,
              "end": 2396
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 2401,
                "end": 2408
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2401,
              "end": 2408
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 2413,
                "end": 2425
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2413,
              "end": 2425
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 2430,
                "end": 2438
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2430,
              "end": 2438
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "yourMembership",
              "loc": {
                "start": 2443,
                "end": 2457
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 2468,
                      "end": 2470
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2468,
                    "end": 2470
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 2479,
                      "end": 2489
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2479,
                    "end": 2489
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 2498,
                      "end": 2508
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2498,
                    "end": 2508
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isAdmin",
                    "loc": {
                      "start": 2517,
                      "end": 2524
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2517,
                    "end": 2524
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "permissions",
                    "loc": {
                      "start": 2533,
                      "end": 2544
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2533,
                    "end": 2544
                  }
                }
              ],
              "loc": {
                "start": 2458,
                "end": 2550
              }
            },
            "loc": {
              "start": 2443,
              "end": 2550
            }
          }
        ],
        "loc": {
          "start": 2319,
          "end": 2552
        }
      },
      "loc": {
        "start": 2315,
        "end": 2552
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 2599,
          "end": 2601
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2599,
        "end": 2601
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bannerImage",
        "loc": {
          "start": 2602,
          "end": 2613
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2602,
        "end": 2613
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 2614,
          "end": 2620
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2614,
        "end": 2620
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "profileImage",
        "loc": {
          "start": 2621,
          "end": 2633
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2621,
        "end": 2633
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 2634,
          "end": 2637
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canAddMembers",
              "loc": {
                "start": 2644,
                "end": 2657
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2644,
              "end": 2657
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 2662,
                "end": 2671
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2662,
              "end": 2671
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 2676,
                "end": 2687
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2676,
              "end": 2687
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReport",
              "loc": {
                "start": 2692,
                "end": 2701
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2692,
              "end": 2701
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 2706,
                "end": 2715
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2706,
              "end": 2715
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 2720,
                "end": 2727
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2720,
              "end": 2727
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 2732,
                "end": 2744
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2732,
              "end": 2744
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 2749,
                "end": 2757
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2749,
              "end": 2757
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "yourMembership",
              "loc": {
                "start": 2762,
                "end": 2776
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 2787,
                      "end": 2789
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2787,
                    "end": 2789
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 2798,
                      "end": 2808
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2798,
                    "end": 2808
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 2817,
                      "end": 2827
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2817,
                    "end": 2827
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isAdmin",
                    "loc": {
                      "start": 2836,
                      "end": 2843
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2836,
                    "end": 2843
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "permissions",
                    "loc": {
                      "start": 2852,
                      "end": 2863
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2852,
                    "end": 2863
                  }
                }
              ],
              "loc": {
                "start": 2777,
                "end": 2869
              }
            },
            "loc": {
              "start": 2762,
              "end": 2869
            }
          }
        ],
        "loc": {
          "start": 2638,
          "end": 2871
        }
      },
      "loc": {
        "start": 2634,
        "end": 2871
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "versions",
        "loc": {
          "start": 2909,
          "end": 2917
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 2924,
                "end": 2936
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 2947,
                      "end": 2949
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2947,
                    "end": 2949
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 2958,
                      "end": 2966
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2958,
                    "end": 2966
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 2975,
                      "end": 2986
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2975,
                    "end": 2986
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 2995,
                      "end": 2999
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2995,
                    "end": 2999
                  }
                }
              ],
              "loc": {
                "start": 2937,
                "end": 3005
              }
            },
            "loc": {
              "start": 2924,
              "end": 3005
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 3010,
                "end": 3012
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3010,
              "end": 3012
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 3017,
                "end": 3027
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3017,
              "end": 3027
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 3032,
                "end": 3042
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3032,
              "end": 3042
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "directoriesCount",
              "loc": {
                "start": 3047,
                "end": 3063
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3047,
              "end": 3063
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLatest",
              "loc": {
                "start": 3068,
                "end": 3076
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3068,
              "end": 3076
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 3081,
                "end": 3090
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3081,
              "end": 3090
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 3095,
                "end": 3107
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3095,
              "end": 3107
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "runProjectsCount",
              "loc": {
                "start": 3112,
                "end": 3128
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3112,
              "end": 3128
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "simplicity",
              "loc": {
                "start": 3133,
                "end": 3143
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3133,
              "end": 3143
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionIndex",
              "loc": {
                "start": 3148,
                "end": 3160
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3148,
              "end": 3160
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionLabel",
              "loc": {
                "start": 3165,
                "end": 3177
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3165,
              "end": 3177
            }
          }
        ],
        "loc": {
          "start": 2918,
          "end": 3179
        }
      },
      "loc": {
        "start": 2909,
        "end": 3179
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 3180,
          "end": 3182
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3180,
        "end": 3182
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 3183,
          "end": 3193
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3183,
        "end": 3193
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 3194,
          "end": 3204
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3194,
        "end": 3204
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 3205,
          "end": 3214
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3205,
        "end": 3214
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "issuesCount",
        "loc": {
          "start": 3215,
          "end": 3226
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3215,
        "end": 3226
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 3227,
          "end": 3233
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Label_list",
              "loc": {
                "start": 3243,
                "end": 3253
              }
            },
            "directives": [],
            "loc": {
              "start": 3240,
              "end": 3253
            }
          }
        ],
        "loc": {
          "start": 3234,
          "end": 3255
        }
      },
      "loc": {
        "start": 3227,
        "end": 3255
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 3256,
          "end": 3261
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Organization",
                "loc": {
                  "start": 3275,
                  "end": 3287
                }
              },
              "loc": {
                "start": 3275,
                "end": 3287
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Organization_nav",
                    "loc": {
                      "start": 3301,
                      "end": 3317
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 3298,
                    "end": 3317
                  }
                }
              ],
              "loc": {
                "start": 3288,
                "end": 3323
              }
            },
            "loc": {
              "start": 3268,
              "end": 3323
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "User",
                "loc": {
                  "start": 3335,
                  "end": 3339
                }
              },
              "loc": {
                "start": 3335,
                "end": 3339
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "User_nav",
                    "loc": {
                      "start": 3353,
                      "end": 3361
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 3350,
                    "end": 3361
                  }
                }
              ],
              "loc": {
                "start": 3340,
                "end": 3367
              }
            },
            "loc": {
              "start": 3328,
              "end": 3367
            }
          }
        ],
        "loc": {
          "start": 3262,
          "end": 3369
        }
      },
      "loc": {
        "start": 3256,
        "end": 3369
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "permissions",
        "loc": {
          "start": 3370,
          "end": 3381
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3370,
        "end": 3381
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "questionsCount",
        "loc": {
          "start": 3382,
          "end": 3396
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3382,
        "end": 3396
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 3397,
          "end": 3402
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3397,
        "end": 3402
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 3403,
          "end": 3412
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3403,
        "end": 3412
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 3413,
          "end": 3417
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Tag_list",
              "loc": {
                "start": 3427,
                "end": 3435
              }
            },
            "directives": [],
            "loc": {
              "start": 3424,
              "end": 3435
            }
          }
        ],
        "loc": {
          "start": 3418,
          "end": 3437
        }
      },
      "loc": {
        "start": 3413,
        "end": 3437
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "transfersCount",
        "loc": {
          "start": 3438,
          "end": 3452
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3438,
        "end": 3452
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "views",
        "loc": {
          "start": 3453,
          "end": 3458
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3453,
        "end": 3458
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 3459,
          "end": 3462
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 3469,
                "end": 3478
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3469,
              "end": 3478
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 3483,
                "end": 3494
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3483,
              "end": 3494
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canTransfer",
              "loc": {
                "start": 3499,
                "end": 3510
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3499,
              "end": 3510
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 3515,
                "end": 3524
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3515,
              "end": 3524
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 3529,
                "end": 3536
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3529,
              "end": 3536
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReact",
              "loc": {
                "start": 3541,
                "end": 3549
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3541,
              "end": 3549
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 3554,
                "end": 3566
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3554,
              "end": 3566
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 3571,
                "end": 3579
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3571,
              "end": 3579
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 3584,
                "end": 3592
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3584,
              "end": 3592
            }
          }
        ],
        "loc": {
          "start": 3463,
          "end": 3594
        }
      },
      "loc": {
        "start": 3459,
        "end": 3594
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 3631,
          "end": 3633
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3631,
        "end": 3633
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 3634,
          "end": 3643
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3634,
        "end": 3643
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "translations",
        "loc": {
          "start": 3683,
          "end": 3695
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 3702,
                "end": 3704
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3702,
              "end": 3704
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "language",
              "loc": {
                "start": 3709,
                "end": 3717
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3709,
              "end": 3717
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "description",
              "loc": {
                "start": 3722,
                "end": 3733
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3722,
              "end": 3733
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 3738,
                "end": 3742
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3738,
              "end": 3742
            }
          }
        ],
        "loc": {
          "start": 3696,
          "end": 3744
        }
      },
      "loc": {
        "start": 3683,
        "end": 3744
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 3745,
          "end": 3747
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3745,
        "end": 3747
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 3748,
          "end": 3758
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3748,
        "end": 3758
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 3759,
          "end": 3769
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3759,
        "end": 3769
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "createdBy",
        "loc": {
          "start": 3770,
          "end": 3779
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 3786,
                "end": 3788
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3786,
              "end": 3788
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bannerImage",
              "loc": {
                "start": 3793,
                "end": 3804
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3793,
              "end": 3804
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 3809,
                "end": 3815
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3809,
              "end": 3815
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBot",
              "loc": {
                "start": 3820,
                "end": 3825
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3820,
              "end": 3825
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 3830,
                "end": 3834
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3830,
              "end": 3834
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "profileImage",
              "loc": {
                "start": 3839,
                "end": 3851
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3839,
              "end": 3851
            }
          }
        ],
        "loc": {
          "start": 3780,
          "end": 3853
        }
      },
      "loc": {
        "start": 3770,
        "end": 3853
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "hasAcceptedAnswer",
        "loc": {
          "start": 3854,
          "end": 3871
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3854,
        "end": 3871
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 3872,
          "end": 3881
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3872,
        "end": 3881
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 3882,
          "end": 3887
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3882,
        "end": 3887
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 3888,
          "end": 3897
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3888,
        "end": 3897
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "answersCount",
        "loc": {
          "start": 3898,
          "end": 3910
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3898,
        "end": 3910
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "commentsCount",
        "loc": {
          "start": 3911,
          "end": 3924
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3911,
        "end": 3924
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "reportsCount",
        "loc": {
          "start": 3925,
          "end": 3937
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3925,
        "end": 3937
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "forObject",
        "loc": {
          "start": 3938,
          "end": 3947
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Api",
                "loc": {
                  "start": 3961,
                  "end": 3964
                }
              },
              "loc": {
                "start": 3961,
                "end": 3964
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Api_nav",
                    "loc": {
                      "start": 3978,
                      "end": 3985
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 3975,
                    "end": 3985
                  }
                }
              ],
              "loc": {
                "start": 3965,
                "end": 3991
              }
            },
            "loc": {
              "start": 3954,
              "end": 3991
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Note",
                "loc": {
                  "start": 4003,
                  "end": 4007
                }
              },
              "loc": {
                "start": 4003,
                "end": 4007
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Note_nav",
                    "loc": {
                      "start": 4021,
                      "end": 4029
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4018,
                    "end": 4029
                  }
                }
              ],
              "loc": {
                "start": 4008,
                "end": 4035
              }
            },
            "loc": {
              "start": 3996,
              "end": 4035
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Organization",
                "loc": {
                  "start": 4047,
                  "end": 4059
                }
              },
              "loc": {
                "start": 4047,
                "end": 4059
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Organization_nav",
                    "loc": {
                      "start": 4073,
                      "end": 4089
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4070,
                    "end": 4089
                  }
                }
              ],
              "loc": {
                "start": 4060,
                "end": 4095
              }
            },
            "loc": {
              "start": 4040,
              "end": 4095
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Project",
                "loc": {
                  "start": 4107,
                  "end": 4114
                }
              },
              "loc": {
                "start": 4107,
                "end": 4114
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Project_nav",
                    "loc": {
                      "start": 4128,
                      "end": 4139
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4125,
                    "end": 4139
                  }
                }
              ],
              "loc": {
                "start": 4115,
                "end": 4145
              }
            },
            "loc": {
              "start": 4100,
              "end": 4145
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Routine",
                "loc": {
                  "start": 4157,
                  "end": 4164
                }
              },
              "loc": {
                "start": 4157,
                "end": 4164
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Routine_nav",
                    "loc": {
                      "start": 4178,
                      "end": 4189
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4175,
                    "end": 4189
                  }
                }
              ],
              "loc": {
                "start": 4165,
                "end": 4195
              }
            },
            "loc": {
              "start": 4150,
              "end": 4195
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "SmartContract",
                "loc": {
                  "start": 4207,
                  "end": 4220
                }
              },
              "loc": {
                "start": 4207,
                "end": 4220
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "SmartContract_nav",
                    "loc": {
                      "start": 4234,
                      "end": 4251
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4231,
                    "end": 4251
                  }
                }
              ],
              "loc": {
                "start": 4221,
                "end": 4257
              }
            },
            "loc": {
              "start": 4200,
              "end": 4257
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Standard",
                "loc": {
                  "start": 4269,
                  "end": 4277
                }
              },
              "loc": {
                "start": 4269,
                "end": 4277
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Standard_nav",
                    "loc": {
                      "start": 4291,
                      "end": 4303
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4288,
                    "end": 4303
                  }
                }
              ],
              "loc": {
                "start": 4278,
                "end": 4309
              }
            },
            "loc": {
              "start": 4262,
              "end": 4309
            }
          }
        ],
        "loc": {
          "start": 3948,
          "end": 4311
        }
      },
      "loc": {
        "start": 3938,
        "end": 4311
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 4312,
          "end": 4316
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Tag_list",
              "loc": {
                "start": 4326,
                "end": 4334
              }
            },
            "directives": [],
            "loc": {
              "start": 4323,
              "end": 4334
            }
          }
        ],
        "loc": {
          "start": 4317,
          "end": 4336
        }
      },
      "loc": {
        "start": 4312,
        "end": 4336
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 4337,
          "end": 4340
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 4347,
                "end": 4355
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4347,
              "end": 4355
            }
          }
        ],
        "loc": {
          "start": 4341,
          "end": 4357
        }
      },
      "loc": {
        "start": 4337,
        "end": 4357
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "versions",
        "loc": {
          "start": 4395,
          "end": 4403
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 4410,
                "end": 4422
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 4433,
                      "end": 4435
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4433,
                    "end": 4435
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 4444,
                      "end": 4452
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4444,
                    "end": 4452
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 4461,
                      "end": 4472
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4461,
                    "end": 4472
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "instructions",
                    "loc": {
                      "start": 4481,
                      "end": 4493
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4481,
                    "end": 4493
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 4502,
                      "end": 4506
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4502,
                    "end": 4506
                  }
                }
              ],
              "loc": {
                "start": 4423,
                "end": 4512
              }
            },
            "loc": {
              "start": 4410,
              "end": 4512
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 4517,
                "end": 4519
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4517,
              "end": 4519
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 4524,
                "end": 4534
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4524,
              "end": 4534
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 4539,
                "end": 4549
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4539,
              "end": 4549
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "completedAt",
              "loc": {
                "start": 4554,
                "end": 4565
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4554,
              "end": 4565
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isAutomatable",
              "loc": {
                "start": 4570,
                "end": 4583
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4570,
              "end": 4583
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isComplete",
              "loc": {
                "start": 4588,
                "end": 4598
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4588,
              "end": 4598
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isDeleted",
              "loc": {
                "start": 4603,
                "end": 4612
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4603,
              "end": 4612
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLatest",
              "loc": {
                "start": 4617,
                "end": 4625
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4617,
              "end": 4625
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 4630,
                "end": 4639
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4630,
              "end": 4639
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "simplicity",
              "loc": {
                "start": 4644,
                "end": 4654
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4644,
              "end": 4654
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timesStarted",
              "loc": {
                "start": 4659,
                "end": 4671
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4659,
              "end": 4671
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timesCompleted",
              "loc": {
                "start": 4676,
                "end": 4690
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4676,
              "end": 4690
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "smartContractCallData",
              "loc": {
                "start": 4695,
                "end": 4716
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4695,
              "end": 4716
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "apiCallData",
              "loc": {
                "start": 4721,
                "end": 4732
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4721,
              "end": 4732
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionIndex",
              "loc": {
                "start": 4737,
                "end": 4749
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4737,
              "end": 4749
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionLabel",
              "loc": {
                "start": 4754,
                "end": 4766
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4754,
              "end": 4766
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "commentsCount",
              "loc": {
                "start": 4771,
                "end": 4784
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4771,
              "end": 4784
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "directoryListingsCount",
              "loc": {
                "start": 4789,
                "end": 4811
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4789,
              "end": 4811
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "forksCount",
              "loc": {
                "start": 4816,
                "end": 4826
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4816,
              "end": 4826
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "inputsCount",
              "loc": {
                "start": 4831,
                "end": 4842
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4831,
              "end": 4842
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "nodesCount",
              "loc": {
                "start": 4847,
                "end": 4857
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4847,
              "end": 4857
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "nodeLinksCount",
              "loc": {
                "start": 4862,
                "end": 4876
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4862,
              "end": 4876
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "outputsCount",
              "loc": {
                "start": 4881,
                "end": 4893
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4881,
              "end": 4893
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 4898,
                "end": 4910
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4898,
              "end": 4910
            }
          }
        ],
        "loc": {
          "start": 4404,
          "end": 4912
        }
      },
      "loc": {
        "start": 4395,
        "end": 4912
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 4913,
          "end": 4915
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4913,
        "end": 4915
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 4916,
          "end": 4926
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4916,
        "end": 4926
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 4927,
          "end": 4937
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4927,
        "end": 4937
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isInternal",
        "loc": {
          "start": 4938,
          "end": 4948
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4938,
        "end": 4948
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 4949,
          "end": 4958
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4949,
        "end": 4958
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "issuesCount",
        "loc": {
          "start": 4959,
          "end": 4970
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4959,
        "end": 4970
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 4971,
          "end": 4977
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Label_list",
              "loc": {
                "start": 4987,
                "end": 4997
              }
            },
            "directives": [],
            "loc": {
              "start": 4984,
              "end": 4997
            }
          }
        ],
        "loc": {
          "start": 4978,
          "end": 4999
        }
      },
      "loc": {
        "start": 4971,
        "end": 4999
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 5000,
          "end": 5005
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Organization",
                "loc": {
                  "start": 5019,
                  "end": 5031
                }
              },
              "loc": {
                "start": 5019,
                "end": 5031
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Organization_nav",
                    "loc": {
                      "start": 5045,
                      "end": 5061
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 5042,
                    "end": 5061
                  }
                }
              ],
              "loc": {
                "start": 5032,
                "end": 5067
              }
            },
            "loc": {
              "start": 5012,
              "end": 5067
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "User",
                "loc": {
                  "start": 5079,
                  "end": 5083
                }
              },
              "loc": {
                "start": 5079,
                "end": 5083
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "User_nav",
                    "loc": {
                      "start": 5097,
                      "end": 5105
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 5094,
                    "end": 5105
                  }
                }
              ],
              "loc": {
                "start": 5084,
                "end": 5111
              }
            },
            "loc": {
              "start": 5072,
              "end": 5111
            }
          }
        ],
        "loc": {
          "start": 5006,
          "end": 5113
        }
      },
      "loc": {
        "start": 5000,
        "end": 5113
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "permissions",
        "loc": {
          "start": 5114,
          "end": 5125
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5114,
        "end": 5125
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "questionsCount",
        "loc": {
          "start": 5126,
          "end": 5140
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5126,
        "end": 5140
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 5141,
          "end": 5146
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5141,
        "end": 5146
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 5147,
          "end": 5156
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5147,
        "end": 5156
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 5157,
          "end": 5161
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Tag_list",
              "loc": {
                "start": 5171,
                "end": 5179
              }
            },
            "directives": [],
            "loc": {
              "start": 5168,
              "end": 5179
            }
          }
        ],
        "loc": {
          "start": 5162,
          "end": 5181
        }
      },
      "loc": {
        "start": 5157,
        "end": 5181
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "transfersCount",
        "loc": {
          "start": 5182,
          "end": 5196
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5182,
        "end": 5196
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "views",
        "loc": {
          "start": 5197,
          "end": 5202
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5197,
        "end": 5202
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 5203,
          "end": 5206
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canComment",
              "loc": {
                "start": 5213,
                "end": 5223
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5213,
              "end": 5223
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 5228,
                "end": 5237
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5228,
              "end": 5237
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 5242,
                "end": 5253
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5242,
              "end": 5253
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 5258,
                "end": 5267
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5258,
              "end": 5267
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 5272,
                "end": 5279
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5272,
              "end": 5279
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReact",
              "loc": {
                "start": 5284,
                "end": 5292
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5284,
              "end": 5292
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 5297,
                "end": 5309
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5297,
              "end": 5309
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 5314,
                "end": 5322
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5314,
              "end": 5322
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 5327,
                "end": 5335
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5327,
              "end": 5335
            }
          }
        ],
        "loc": {
          "start": 5207,
          "end": 5337
        }
      },
      "loc": {
        "start": 5203,
        "end": 5337
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 5374,
          "end": 5376
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5374,
        "end": 5376
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isInternal",
        "loc": {
          "start": 5377,
          "end": 5387
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5377,
        "end": 5387
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 5388,
          "end": 5397
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5388,
        "end": 5397
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "versions",
        "loc": {
          "start": 5447,
          "end": 5455
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 5462,
                "end": 5474
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 5485,
                      "end": 5487
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5485,
                    "end": 5487
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 5496,
                      "end": 5504
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5496,
                    "end": 5504
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 5513,
                      "end": 5524
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5513,
                    "end": 5524
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "jsonVariable",
                    "loc": {
                      "start": 5533,
                      "end": 5545
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5533,
                    "end": 5545
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 5554,
                      "end": 5558
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5554,
                    "end": 5558
                  }
                }
              ],
              "loc": {
                "start": 5475,
                "end": 5564
              }
            },
            "loc": {
              "start": 5462,
              "end": 5564
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 5569,
                "end": 5571
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5569,
              "end": 5571
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 5576,
                "end": 5586
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5576,
              "end": 5586
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 5591,
                "end": 5601
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5591,
              "end": 5601
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isComplete",
              "loc": {
                "start": 5606,
                "end": 5616
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5606,
              "end": 5616
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isDeleted",
              "loc": {
                "start": 5621,
                "end": 5630
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5621,
              "end": 5630
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLatest",
              "loc": {
                "start": 5635,
                "end": 5643
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5635,
              "end": 5643
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 5648,
                "end": 5657
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5648,
              "end": 5657
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "default",
              "loc": {
                "start": 5662,
                "end": 5669
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5662,
              "end": 5669
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "contractType",
              "loc": {
                "start": 5674,
                "end": 5686
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5674,
              "end": 5686
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "content",
              "loc": {
                "start": 5691,
                "end": 5698
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5691,
              "end": 5698
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionIndex",
              "loc": {
                "start": 5703,
                "end": 5715
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5703,
              "end": 5715
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionLabel",
              "loc": {
                "start": 5720,
                "end": 5732
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5720,
              "end": 5732
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "commentsCount",
              "loc": {
                "start": 5737,
                "end": 5750
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5737,
              "end": 5750
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "directoryListingsCount",
              "loc": {
                "start": 5755,
                "end": 5777
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5755,
              "end": 5777
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "forksCount",
              "loc": {
                "start": 5782,
                "end": 5792
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5782,
              "end": 5792
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 5797,
                "end": 5809
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5797,
              "end": 5809
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 5814,
                "end": 5817
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canComment",
                    "loc": {
                      "start": 5828,
                      "end": 5838
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5828,
                    "end": 5838
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canCopy",
                    "loc": {
                      "start": 5847,
                      "end": 5854
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5847,
                    "end": 5854
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 5863,
                      "end": 5872
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5863,
                    "end": 5872
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 5881,
                      "end": 5890
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5881,
                    "end": 5890
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 5899,
                      "end": 5908
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5899,
                    "end": 5908
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUse",
                    "loc": {
                      "start": 5917,
                      "end": 5923
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5917,
                    "end": 5923
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 5932,
                      "end": 5939
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5932,
                    "end": 5939
                  }
                }
              ],
              "loc": {
                "start": 5818,
                "end": 5945
              }
            },
            "loc": {
              "start": 5814,
              "end": 5945
            }
          }
        ],
        "loc": {
          "start": 5456,
          "end": 5947
        }
      },
      "loc": {
        "start": 5447,
        "end": 5947
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 5948,
          "end": 5950
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5948,
        "end": 5950
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 5951,
          "end": 5961
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5951,
        "end": 5961
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 5962,
          "end": 5972
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5962,
        "end": 5972
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 5973,
          "end": 5982
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5973,
        "end": 5982
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "issuesCount",
        "loc": {
          "start": 5983,
          "end": 5994
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5983,
        "end": 5994
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 5995,
          "end": 6001
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Label_list",
              "loc": {
                "start": 6011,
                "end": 6021
              }
            },
            "directives": [],
            "loc": {
              "start": 6008,
              "end": 6021
            }
          }
        ],
        "loc": {
          "start": 6002,
          "end": 6023
        }
      },
      "loc": {
        "start": 5995,
        "end": 6023
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 6024,
          "end": 6029
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Organization",
                "loc": {
                  "start": 6043,
                  "end": 6055
                }
              },
              "loc": {
                "start": 6043,
                "end": 6055
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Organization_nav",
                    "loc": {
                      "start": 6069,
                      "end": 6085
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 6066,
                    "end": 6085
                  }
                }
              ],
              "loc": {
                "start": 6056,
                "end": 6091
              }
            },
            "loc": {
              "start": 6036,
              "end": 6091
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "User",
                "loc": {
                  "start": 6103,
                  "end": 6107
                }
              },
              "loc": {
                "start": 6103,
                "end": 6107
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "User_nav",
                    "loc": {
                      "start": 6121,
                      "end": 6129
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 6118,
                    "end": 6129
                  }
                }
              ],
              "loc": {
                "start": 6108,
                "end": 6135
              }
            },
            "loc": {
              "start": 6096,
              "end": 6135
            }
          }
        ],
        "loc": {
          "start": 6030,
          "end": 6137
        }
      },
      "loc": {
        "start": 6024,
        "end": 6137
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "permissions",
        "loc": {
          "start": 6138,
          "end": 6149
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6138,
        "end": 6149
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "questionsCount",
        "loc": {
          "start": 6150,
          "end": 6164
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6150,
        "end": 6164
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 6165,
          "end": 6170
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6165,
        "end": 6170
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 6171,
          "end": 6180
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6171,
        "end": 6180
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 6181,
          "end": 6185
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Tag_list",
              "loc": {
                "start": 6195,
                "end": 6203
              }
            },
            "directives": [],
            "loc": {
              "start": 6192,
              "end": 6203
            }
          }
        ],
        "loc": {
          "start": 6186,
          "end": 6205
        }
      },
      "loc": {
        "start": 6181,
        "end": 6205
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "transfersCount",
        "loc": {
          "start": 6206,
          "end": 6220
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6206,
        "end": 6220
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "views",
        "loc": {
          "start": 6221,
          "end": 6226
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6221,
        "end": 6226
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 6227,
          "end": 6230
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 6237,
                "end": 6246
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6237,
              "end": 6246
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 6251,
                "end": 6262
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6251,
              "end": 6262
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canTransfer",
              "loc": {
                "start": 6267,
                "end": 6278
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6267,
              "end": 6278
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 6283,
                "end": 6292
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6283,
              "end": 6292
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 6297,
                "end": 6304
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6297,
              "end": 6304
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReact",
              "loc": {
                "start": 6309,
                "end": 6317
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6309,
              "end": 6317
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 6322,
                "end": 6334
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6322,
              "end": 6334
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 6339,
                "end": 6347
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6339,
              "end": 6347
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 6352,
                "end": 6360
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6352,
              "end": 6360
            }
          }
        ],
        "loc": {
          "start": 6231,
          "end": 6362
        }
      },
      "loc": {
        "start": 6227,
        "end": 6362
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 6411,
          "end": 6413
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6411,
        "end": 6413
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 6414,
          "end": 6423
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6414,
        "end": 6423
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "versions",
        "loc": {
          "start": 6463,
          "end": 6471
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 6478,
                "end": 6490
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 6501,
                      "end": 6503
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6501,
                    "end": 6503
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 6512,
                      "end": 6520
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6512,
                    "end": 6520
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 6529,
                      "end": 6540
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6529,
                    "end": 6540
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "jsonVariable",
                    "loc": {
                      "start": 6549,
                      "end": 6561
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6549,
                    "end": 6561
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 6570,
                      "end": 6574
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6570,
                    "end": 6574
                  }
                }
              ],
              "loc": {
                "start": 6491,
                "end": 6580
              }
            },
            "loc": {
              "start": 6478,
              "end": 6580
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 6585,
                "end": 6587
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6585,
              "end": 6587
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 6592,
                "end": 6602
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6592,
              "end": 6602
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 6607,
                "end": 6617
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6607,
              "end": 6617
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isComplete",
              "loc": {
                "start": 6622,
                "end": 6632
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6622,
              "end": 6632
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isFile",
              "loc": {
                "start": 6637,
                "end": 6643
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6637,
              "end": 6643
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLatest",
              "loc": {
                "start": 6648,
                "end": 6656
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6648,
              "end": 6656
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 6661,
                "end": 6670
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6661,
              "end": 6670
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "default",
              "loc": {
                "start": 6675,
                "end": 6682
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6675,
              "end": 6682
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "standardType",
              "loc": {
                "start": 6687,
                "end": 6699
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6687,
              "end": 6699
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "props",
              "loc": {
                "start": 6704,
                "end": 6709
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6704,
              "end": 6709
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "yup",
              "loc": {
                "start": 6714,
                "end": 6717
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6714,
              "end": 6717
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionIndex",
              "loc": {
                "start": 6722,
                "end": 6734
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6722,
              "end": 6734
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionLabel",
              "loc": {
                "start": 6739,
                "end": 6751
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6739,
              "end": 6751
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "commentsCount",
              "loc": {
                "start": 6756,
                "end": 6769
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6756,
              "end": 6769
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "directoryListingsCount",
              "loc": {
                "start": 6774,
                "end": 6796
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6774,
              "end": 6796
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "forksCount",
              "loc": {
                "start": 6801,
                "end": 6811
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6801,
              "end": 6811
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 6816,
                "end": 6828
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6816,
              "end": 6828
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 6833,
                "end": 6836
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canComment",
                    "loc": {
                      "start": 6847,
                      "end": 6857
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6847,
                    "end": 6857
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canCopy",
                    "loc": {
                      "start": 6866,
                      "end": 6873
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6866,
                    "end": 6873
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 6882,
                      "end": 6891
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6882,
                    "end": 6891
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 6900,
                      "end": 6909
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6900,
                    "end": 6909
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 6918,
                      "end": 6927
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6918,
                    "end": 6927
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUse",
                    "loc": {
                      "start": 6936,
                      "end": 6942
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6936,
                    "end": 6942
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 6951,
                      "end": 6958
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6951,
                    "end": 6958
                  }
                }
              ],
              "loc": {
                "start": 6837,
                "end": 6964
              }
            },
            "loc": {
              "start": 6833,
              "end": 6964
            }
          }
        ],
        "loc": {
          "start": 6472,
          "end": 6966
        }
      },
      "loc": {
        "start": 6463,
        "end": 6966
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 6967,
          "end": 6969
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6967,
        "end": 6969
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 6970,
          "end": 6980
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6970,
        "end": 6980
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 6981,
          "end": 6991
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6981,
        "end": 6991
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 6992,
          "end": 7001
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6992,
        "end": 7001
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "issuesCount",
        "loc": {
          "start": 7002,
          "end": 7013
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7002,
        "end": 7013
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 7014,
          "end": 7020
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Label_list",
              "loc": {
                "start": 7030,
                "end": 7040
              }
            },
            "directives": [],
            "loc": {
              "start": 7027,
              "end": 7040
            }
          }
        ],
        "loc": {
          "start": 7021,
          "end": 7042
        }
      },
      "loc": {
        "start": 7014,
        "end": 7042
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 7043,
          "end": 7048
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Organization",
                "loc": {
                  "start": 7062,
                  "end": 7074
                }
              },
              "loc": {
                "start": 7062,
                "end": 7074
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Organization_nav",
                    "loc": {
                      "start": 7088,
                      "end": 7104
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 7085,
                    "end": 7104
                  }
                }
              ],
              "loc": {
                "start": 7075,
                "end": 7110
              }
            },
            "loc": {
              "start": 7055,
              "end": 7110
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "User",
                "loc": {
                  "start": 7122,
                  "end": 7126
                }
              },
              "loc": {
                "start": 7122,
                "end": 7126
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "User_nav",
                    "loc": {
                      "start": 7140,
                      "end": 7148
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 7137,
                    "end": 7148
                  }
                }
              ],
              "loc": {
                "start": 7127,
                "end": 7154
              }
            },
            "loc": {
              "start": 7115,
              "end": 7154
            }
          }
        ],
        "loc": {
          "start": 7049,
          "end": 7156
        }
      },
      "loc": {
        "start": 7043,
        "end": 7156
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "permissions",
        "loc": {
          "start": 7157,
          "end": 7168
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7157,
        "end": 7168
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "questionsCount",
        "loc": {
          "start": 7169,
          "end": 7183
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7169,
        "end": 7183
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 7184,
          "end": 7189
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7184,
        "end": 7189
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 7190,
          "end": 7199
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7190,
        "end": 7199
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 7200,
          "end": 7204
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Tag_list",
              "loc": {
                "start": 7214,
                "end": 7222
              }
            },
            "directives": [],
            "loc": {
              "start": 7211,
              "end": 7222
            }
          }
        ],
        "loc": {
          "start": 7205,
          "end": 7224
        }
      },
      "loc": {
        "start": 7200,
        "end": 7224
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "transfersCount",
        "loc": {
          "start": 7225,
          "end": 7239
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7225,
        "end": 7239
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "views",
        "loc": {
          "start": 7240,
          "end": 7245
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7240,
        "end": 7245
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 7246,
          "end": 7249
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 7256,
                "end": 7265
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7256,
              "end": 7265
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 7270,
                "end": 7281
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7270,
              "end": 7281
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canTransfer",
              "loc": {
                "start": 7286,
                "end": 7297
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7286,
              "end": 7297
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 7302,
                "end": 7311
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7302,
              "end": 7311
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 7316,
                "end": 7323
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7316,
              "end": 7323
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReact",
              "loc": {
                "start": 7328,
                "end": 7336
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7328,
              "end": 7336
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 7341,
                "end": 7353
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7341,
              "end": 7353
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 7358,
                "end": 7366
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7358,
              "end": 7366
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 7371,
                "end": 7379
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7371,
              "end": 7379
            }
          }
        ],
        "loc": {
          "start": 7250,
          "end": 7381
        }
      },
      "loc": {
        "start": 7246,
        "end": 7381
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 7420,
          "end": 7422
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7420,
        "end": 7422
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 7423,
          "end": 7432
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7423,
        "end": 7432
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 7462,
          "end": 7464
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7462,
        "end": 7464
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 7465,
          "end": 7475
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7465,
        "end": 7475
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tag",
        "loc": {
          "start": 7476,
          "end": 7479
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7476,
        "end": 7479
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 7480,
          "end": 7489
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7480,
        "end": 7489
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "translations",
        "loc": {
          "start": 7490,
          "end": 7502
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 7509,
                "end": 7511
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7509,
              "end": 7511
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "language",
              "loc": {
                "start": 7516,
                "end": 7524
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7516,
              "end": 7524
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "description",
              "loc": {
                "start": 7529,
                "end": 7540
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7529,
              "end": 7540
            }
          }
        ],
        "loc": {
          "start": 7503,
          "end": 7542
        }
      },
      "loc": {
        "start": 7490,
        "end": 7542
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 7543,
          "end": 7546
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isOwn",
              "loc": {
                "start": 7553,
                "end": 7558
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7553,
              "end": 7558
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 7563,
                "end": 7575
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7563,
              "end": 7575
            }
          }
        ],
        "loc": {
          "start": 7547,
          "end": 7577
        }
      },
      "loc": {
        "start": 7543,
        "end": 7577
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "translations",
        "loc": {
          "start": 7609,
          "end": 7621
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 7628,
                "end": 7630
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7628,
              "end": 7630
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "language",
              "loc": {
                "start": 7635,
                "end": 7643
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7635,
              "end": 7643
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bio",
              "loc": {
                "start": 7648,
                "end": 7651
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7648,
              "end": 7651
            }
          }
        ],
        "loc": {
          "start": 7622,
          "end": 7653
        }
      },
      "loc": {
        "start": 7609,
        "end": 7653
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 7654,
          "end": 7656
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7654,
        "end": 7656
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 7657,
          "end": 7667
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7657,
        "end": 7667
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bannerImage",
        "loc": {
          "start": 7668,
          "end": 7679
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7668,
        "end": 7679
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 7680,
          "end": 7686
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7680,
        "end": 7686
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isBot",
        "loc": {
          "start": 7687,
          "end": 7692
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7687,
        "end": 7692
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "name",
        "loc": {
          "start": 7693,
          "end": 7697
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7693,
        "end": 7697
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "profileImage",
        "loc": {
          "start": 7698,
          "end": 7710
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7698,
        "end": 7710
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 7711,
          "end": 7720
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7711,
        "end": 7720
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "reportsReceivedCount",
        "loc": {
          "start": 7721,
          "end": 7741
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7721,
        "end": 7741
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 7742,
          "end": 7745
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 7752,
                "end": 7761
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7752,
              "end": 7761
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReport",
              "loc": {
                "start": 7766,
                "end": 7775
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7766,
              "end": 7775
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 7780,
                "end": 7789
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7780,
              "end": 7789
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 7794,
                "end": 7806
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7794,
              "end": 7806
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 7811,
                "end": 7819
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7811,
              "end": 7819
            }
          }
        ],
        "loc": {
          "start": 7746,
          "end": 7821
        }
      },
      "loc": {
        "start": 7742,
        "end": 7821
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 7852,
          "end": 7854
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7852,
        "end": 7854
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bannerImage",
        "loc": {
          "start": 7855,
          "end": 7866
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7855,
        "end": 7866
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 7867,
          "end": 7873
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7867,
        "end": 7873
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isBot",
        "loc": {
          "start": 7874,
          "end": 7879
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7874,
        "end": 7879
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "name",
        "loc": {
          "start": 7880,
          "end": 7884
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7880,
        "end": 7884
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "profileImage",
        "loc": {
          "start": 7885,
          "end": 7897
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7885,
        "end": 7897
      }
    }
  ],
  "returnType": null,
  "parentType": null,
  "schema": null,
  "fragments": {
    "Api_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Api_list",
        "loc": {
          "start": 10,
          "end": 18
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Api",
          "loc": {
            "start": 22,
            "end": 25
          }
        },
        "loc": {
          "start": 22,
          "end": 25
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versions",
              "loc": {
                "start": 28,
                "end": 36
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 43,
                      "end": 55
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 66,
                            "end": 68
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 66,
                          "end": 68
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 77,
                            "end": 85
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 77,
                          "end": 85
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "details",
                          "loc": {
                            "start": 94,
                            "end": 101
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 94,
                          "end": 101
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 110,
                            "end": 114
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 110,
                          "end": 114
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "summary",
                          "loc": {
                            "start": 123,
                            "end": 130
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 123,
                          "end": 130
                        }
                      }
                    ],
                    "loc": {
                      "start": 56,
                      "end": 136
                    }
                  },
                  "loc": {
                    "start": 43,
                    "end": 136
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 141,
                      "end": 143
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 141,
                    "end": 143
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 148,
                      "end": 158
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 148,
                    "end": 158
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 163,
                      "end": 173
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 163,
                    "end": 173
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "callLink",
                    "loc": {
                      "start": 178,
                      "end": 186
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 178,
                    "end": 186
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "commentsCount",
                    "loc": {
                      "start": 191,
                      "end": 204
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 191,
                    "end": 204
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "documentationLink",
                    "loc": {
                      "start": 209,
                      "end": 226
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 209,
                    "end": 226
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "forksCount",
                    "loc": {
                      "start": 231,
                      "end": 241
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 231,
                    "end": 241
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 246,
                      "end": 254
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 246,
                    "end": 254
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 259,
                      "end": 268
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 259,
                    "end": 268
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 273,
                      "end": 285
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 273,
                    "end": 285
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 290,
                      "end": 302
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 290,
                    "end": 302
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 307,
                      "end": 319
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 307,
                    "end": 319
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 324,
                      "end": 327
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canComment",
                          "loc": {
                            "start": 338,
                            "end": 348
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 338,
                          "end": 348
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canCopy",
                          "loc": {
                            "start": 357,
                            "end": 364
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 357,
                          "end": 364
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canDelete",
                          "loc": {
                            "start": 373,
                            "end": 382
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 373,
                          "end": 382
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canReport",
                          "loc": {
                            "start": 391,
                            "end": 400
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 391,
                          "end": 400
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 409,
                            "end": 418
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 409,
                          "end": 418
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUse",
                          "loc": {
                            "start": 427,
                            "end": 433
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 427,
                          "end": 433
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 442,
                            "end": 449
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 442,
                          "end": 449
                        }
                      }
                    ],
                    "loc": {
                      "start": 328,
                      "end": 455
                    }
                  },
                  "loc": {
                    "start": 324,
                    "end": 455
                  }
                }
              ],
              "loc": {
                "start": 37,
                "end": 457
              }
            },
            "loc": {
              "start": 28,
              "end": 457
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 458,
                "end": 460
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 458,
              "end": 460
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 461,
                "end": 471
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 461,
              "end": 471
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 472,
                "end": 482
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 472,
              "end": 482
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 483,
                "end": 492
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 483,
              "end": 492
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 493,
                "end": 504
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 493,
              "end": 504
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 505,
                "end": 511
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Label_list",
                    "loc": {
                      "start": 521,
                      "end": 531
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 518,
                    "end": 531
                  }
                }
              ],
              "loc": {
                "start": 512,
                "end": 533
              }
            },
            "loc": {
              "start": 505,
              "end": 533
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 534,
                "end": 539
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Organization",
                      "loc": {
                        "start": 553,
                        "end": 565
                      }
                    },
                    "loc": {
                      "start": 553,
                      "end": 565
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Organization_nav",
                          "loc": {
                            "start": 579,
                            "end": 595
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 576,
                          "end": 595
                        }
                      }
                    ],
                    "loc": {
                      "start": 566,
                      "end": 601
                    }
                  },
                  "loc": {
                    "start": 546,
                    "end": 601
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "User",
                      "loc": {
                        "start": 613,
                        "end": 617
                      }
                    },
                    "loc": {
                      "start": 613,
                      "end": 617
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "User_nav",
                          "loc": {
                            "start": 631,
                            "end": 639
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 628,
                          "end": 639
                        }
                      }
                    ],
                    "loc": {
                      "start": 618,
                      "end": 645
                    }
                  },
                  "loc": {
                    "start": 606,
                    "end": 645
                  }
                }
              ],
              "loc": {
                "start": 540,
                "end": 647
              }
            },
            "loc": {
              "start": 534,
              "end": 647
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 648,
                "end": 659
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 648,
              "end": 659
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 660,
                "end": 674
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 660,
              "end": 674
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 675,
                "end": 680
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 675,
              "end": 680
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 681,
                "end": 690
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 681,
              "end": 690
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 691,
                "end": 695
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Tag_list",
                    "loc": {
                      "start": 705,
                      "end": 713
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 702,
                    "end": 713
                  }
                }
              ],
              "loc": {
                "start": 696,
                "end": 715
              }
            },
            "loc": {
              "start": 691,
              "end": 715
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 716,
                "end": 730
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 716,
              "end": 730
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 731,
                "end": 736
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 731,
              "end": 736
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 737,
                "end": 740
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 747,
                      "end": 756
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 747,
                    "end": 756
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 761,
                      "end": 772
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 761,
                    "end": 772
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canTransfer",
                    "loc": {
                      "start": 777,
                      "end": 788
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 777,
                    "end": 788
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 793,
                      "end": 802
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 793,
                    "end": 802
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 807,
                      "end": 814
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 807,
                    "end": 814
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 819,
                      "end": 827
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 819,
                    "end": 827
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 832,
                      "end": 844
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 832,
                    "end": 844
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 849,
                      "end": 857
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 849,
                    "end": 857
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 862,
                      "end": 870
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 862,
                    "end": 870
                  }
                }
              ],
              "loc": {
                "start": 741,
                "end": 872
              }
            },
            "loc": {
              "start": 737,
              "end": 872
            }
          }
        ],
        "loc": {
          "start": 26,
          "end": 874
        }
      },
      "loc": {
        "start": 1,
        "end": 874
      }
    },
    "Api_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Api_nav",
        "loc": {
          "start": 884,
          "end": 891
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Api",
          "loc": {
            "start": 895,
            "end": 898
          }
        },
        "loc": {
          "start": 895,
          "end": 898
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 901,
                "end": 903
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 901,
              "end": 903
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 904,
                "end": 913
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 904,
              "end": 913
            }
          }
        ],
        "loc": {
          "start": 899,
          "end": 915
        }
      },
      "loc": {
        "start": 875,
        "end": 915
      }
    },
    "Label_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Label_list",
        "loc": {
          "start": 925,
          "end": 935
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Label",
          "loc": {
            "start": 939,
            "end": 944
          }
        },
        "loc": {
          "start": 939,
          "end": 944
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 947,
                "end": 949
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 947,
              "end": 949
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 950,
                "end": 960
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 950,
              "end": 960
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 961,
                "end": 971
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 961,
              "end": 971
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "color",
              "loc": {
                "start": 972,
                "end": 977
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 972,
              "end": 977
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "label",
              "loc": {
                "start": 978,
                "end": 983
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 978,
              "end": 983
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 984,
                "end": 989
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Organization",
                      "loc": {
                        "start": 1003,
                        "end": 1015
                      }
                    },
                    "loc": {
                      "start": 1003,
                      "end": 1015
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Organization_nav",
                          "loc": {
                            "start": 1029,
                            "end": 1045
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 1026,
                          "end": 1045
                        }
                      }
                    ],
                    "loc": {
                      "start": 1016,
                      "end": 1051
                    }
                  },
                  "loc": {
                    "start": 996,
                    "end": 1051
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "User",
                      "loc": {
                        "start": 1063,
                        "end": 1067
                      }
                    },
                    "loc": {
                      "start": 1063,
                      "end": 1067
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "User_nav",
                          "loc": {
                            "start": 1081,
                            "end": 1089
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 1078,
                          "end": 1089
                        }
                      }
                    ],
                    "loc": {
                      "start": 1068,
                      "end": 1095
                    }
                  },
                  "loc": {
                    "start": 1056,
                    "end": 1095
                  }
                }
              ],
              "loc": {
                "start": 990,
                "end": 1097
              }
            },
            "loc": {
              "start": 984,
              "end": 1097
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 1098,
                "end": 1101
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 1108,
                      "end": 1117
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1108,
                    "end": 1117
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 1122,
                      "end": 1131
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1122,
                    "end": 1131
                  }
                }
              ],
              "loc": {
                "start": 1102,
                "end": 1133
              }
            },
            "loc": {
              "start": 1098,
              "end": 1133
            }
          }
        ],
        "loc": {
          "start": 945,
          "end": 1135
        }
      },
      "loc": {
        "start": 916,
        "end": 1135
      }
    },
    "Note_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Note_list",
        "loc": {
          "start": 1145,
          "end": 1154
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Note",
          "loc": {
            "start": 1158,
            "end": 1162
          }
        },
        "loc": {
          "start": 1158,
          "end": 1162
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versions",
              "loc": {
                "start": 1165,
                "end": 1173
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 1180,
                      "end": 1192
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 1203,
                            "end": 1205
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1203,
                          "end": 1205
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 1214,
                            "end": 1222
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1214,
                          "end": 1222
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 1231,
                            "end": 1242
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1231,
                          "end": 1242
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 1251,
                            "end": 1255
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1251,
                          "end": 1255
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "pages",
                          "loc": {
                            "start": 1264,
                            "end": 1269
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 1284,
                                  "end": 1286
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 1284,
                                "end": 1286
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "pageIndex",
                                "loc": {
                                  "start": 1299,
                                  "end": 1308
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 1299,
                                "end": 1308
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "text",
                                "loc": {
                                  "start": 1321,
                                  "end": 1325
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 1321,
                                "end": 1325
                              }
                            }
                          ],
                          "loc": {
                            "start": 1270,
                            "end": 1335
                          }
                        },
                        "loc": {
                          "start": 1264,
                          "end": 1335
                        }
                      }
                    ],
                    "loc": {
                      "start": 1193,
                      "end": 1341
                    }
                  },
                  "loc": {
                    "start": 1180,
                    "end": 1341
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 1346,
                      "end": 1348
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1346,
                    "end": 1348
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 1353,
                      "end": 1363
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1353,
                    "end": 1363
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 1368,
                      "end": 1378
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1368,
                    "end": 1378
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 1383,
                      "end": 1391
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1383,
                    "end": 1391
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 1396,
                      "end": 1405
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1396,
                    "end": 1405
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 1410,
                      "end": 1422
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1410,
                    "end": 1422
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 1427,
                      "end": 1439
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1427,
                    "end": 1439
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 1444,
                      "end": 1456
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1444,
                    "end": 1456
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 1461,
                      "end": 1464
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canComment",
                          "loc": {
                            "start": 1475,
                            "end": 1485
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1475,
                          "end": 1485
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canCopy",
                          "loc": {
                            "start": 1494,
                            "end": 1501
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1494,
                          "end": 1501
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canDelete",
                          "loc": {
                            "start": 1510,
                            "end": 1519
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1510,
                          "end": 1519
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canReport",
                          "loc": {
                            "start": 1528,
                            "end": 1537
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1528,
                          "end": 1537
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 1546,
                            "end": 1555
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1546,
                          "end": 1555
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUse",
                          "loc": {
                            "start": 1564,
                            "end": 1570
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1564,
                          "end": 1570
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 1579,
                            "end": 1586
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1579,
                          "end": 1586
                        }
                      }
                    ],
                    "loc": {
                      "start": 1465,
                      "end": 1592
                    }
                  },
                  "loc": {
                    "start": 1461,
                    "end": 1592
                  }
                }
              ],
              "loc": {
                "start": 1174,
                "end": 1594
              }
            },
            "loc": {
              "start": 1165,
              "end": 1594
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 1595,
                "end": 1597
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1595,
              "end": 1597
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 1598,
                "end": 1608
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1598,
              "end": 1608
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 1609,
                "end": 1619
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1609,
              "end": 1619
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 1620,
                "end": 1629
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1620,
              "end": 1629
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 1630,
                "end": 1641
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1630,
              "end": 1641
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 1642,
                "end": 1648
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Label_list",
                    "loc": {
                      "start": 1658,
                      "end": 1668
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1655,
                    "end": 1668
                  }
                }
              ],
              "loc": {
                "start": 1649,
                "end": 1670
              }
            },
            "loc": {
              "start": 1642,
              "end": 1670
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 1671,
                "end": 1676
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Organization",
                      "loc": {
                        "start": 1690,
                        "end": 1702
                      }
                    },
                    "loc": {
                      "start": 1690,
                      "end": 1702
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Organization_nav",
                          "loc": {
                            "start": 1716,
                            "end": 1732
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 1713,
                          "end": 1732
                        }
                      }
                    ],
                    "loc": {
                      "start": 1703,
                      "end": 1738
                    }
                  },
                  "loc": {
                    "start": 1683,
                    "end": 1738
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "User",
                      "loc": {
                        "start": 1750,
                        "end": 1754
                      }
                    },
                    "loc": {
                      "start": 1750,
                      "end": 1754
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "User_nav",
                          "loc": {
                            "start": 1768,
                            "end": 1776
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 1765,
                          "end": 1776
                        }
                      }
                    ],
                    "loc": {
                      "start": 1755,
                      "end": 1782
                    }
                  },
                  "loc": {
                    "start": 1743,
                    "end": 1782
                  }
                }
              ],
              "loc": {
                "start": 1677,
                "end": 1784
              }
            },
            "loc": {
              "start": 1671,
              "end": 1784
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 1785,
                "end": 1796
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1785,
              "end": 1796
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 1797,
                "end": 1811
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1797,
              "end": 1811
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 1812,
                "end": 1817
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1812,
              "end": 1817
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 1818,
                "end": 1827
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1818,
              "end": 1827
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 1828,
                "end": 1832
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Tag_list",
                    "loc": {
                      "start": 1842,
                      "end": 1850
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1839,
                    "end": 1850
                  }
                }
              ],
              "loc": {
                "start": 1833,
                "end": 1852
              }
            },
            "loc": {
              "start": 1828,
              "end": 1852
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 1853,
                "end": 1867
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1853,
              "end": 1867
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 1868,
                "end": 1873
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1868,
              "end": 1873
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 1874,
                "end": 1877
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 1884,
                      "end": 1893
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1884,
                    "end": 1893
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 1898,
                      "end": 1909
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1898,
                    "end": 1909
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canTransfer",
                    "loc": {
                      "start": 1914,
                      "end": 1925
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1914,
                    "end": 1925
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 1930,
                      "end": 1939
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1930,
                    "end": 1939
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 1944,
                      "end": 1951
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1944,
                    "end": 1951
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 1956,
                      "end": 1964
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1956,
                    "end": 1964
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 1969,
                      "end": 1981
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1969,
                    "end": 1981
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 1986,
                      "end": 1994
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1986,
                    "end": 1994
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 1999,
                      "end": 2007
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1999,
                    "end": 2007
                  }
                }
              ],
              "loc": {
                "start": 1878,
                "end": 2009
              }
            },
            "loc": {
              "start": 1874,
              "end": 2009
            }
          }
        ],
        "loc": {
          "start": 1163,
          "end": 2011
        }
      },
      "loc": {
        "start": 1136,
        "end": 2011
      }
    },
    "Note_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Note_nav",
        "loc": {
          "start": 2021,
          "end": 2029
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Note",
          "loc": {
            "start": 2033,
            "end": 2037
          }
        },
        "loc": {
          "start": 2033,
          "end": 2037
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 2040,
                "end": 2042
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2040,
              "end": 2042
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 2043,
                "end": 2052
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2043,
              "end": 2052
            }
          }
        ],
        "loc": {
          "start": 2038,
          "end": 2054
        }
      },
      "loc": {
        "start": 2012,
        "end": 2054
      }
    },
    "Organization_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Organization_list",
        "loc": {
          "start": 2064,
          "end": 2081
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Organization",
          "loc": {
            "start": 2085,
            "end": 2097
          }
        },
        "loc": {
          "start": 2085,
          "end": 2097
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 2100,
                "end": 2102
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2100,
              "end": 2102
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bannerImage",
              "loc": {
                "start": 2103,
                "end": 2114
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2103,
              "end": 2114
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 2115,
                "end": 2121
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2115,
              "end": 2121
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 2122,
                "end": 2132
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2122,
              "end": 2132
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 2133,
                "end": 2143
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2133,
              "end": 2143
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isOpenToNewMembers",
              "loc": {
                "start": 2144,
                "end": 2162
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2144,
              "end": 2162
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 2163,
                "end": 2172
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2163,
              "end": 2172
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "commentsCount",
              "loc": {
                "start": 2173,
                "end": 2186
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2173,
              "end": 2186
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "membersCount",
              "loc": {
                "start": 2187,
                "end": 2199
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2187,
              "end": 2199
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "profileImage",
              "loc": {
                "start": 2200,
                "end": 2212
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2200,
              "end": 2212
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 2213,
                "end": 2225
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2213,
              "end": 2225
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 2226,
                "end": 2235
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2226,
              "end": 2235
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 2236,
                "end": 2240
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Tag_list",
                    "loc": {
                      "start": 2250,
                      "end": 2258
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 2247,
                    "end": 2258
                  }
                }
              ],
              "loc": {
                "start": 2241,
                "end": 2260
              }
            },
            "loc": {
              "start": 2236,
              "end": 2260
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 2261,
                "end": 2273
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 2280,
                      "end": 2282
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2280,
                    "end": 2282
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 2287,
                      "end": 2295
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2287,
                    "end": 2295
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "bio",
                    "loc": {
                      "start": 2300,
                      "end": 2303
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2300,
                    "end": 2303
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 2308,
                      "end": 2312
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2308,
                    "end": 2312
                  }
                }
              ],
              "loc": {
                "start": 2274,
                "end": 2314
              }
            },
            "loc": {
              "start": 2261,
              "end": 2314
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 2315,
                "end": 2318
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canAddMembers",
                    "loc": {
                      "start": 2325,
                      "end": 2338
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2325,
                    "end": 2338
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 2343,
                      "end": 2352
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2343,
                    "end": 2352
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 2357,
                      "end": 2368
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2357,
                    "end": 2368
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 2373,
                      "end": 2382
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2373,
                    "end": 2382
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 2387,
                      "end": 2396
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2387,
                    "end": 2396
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 2401,
                      "end": 2408
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2401,
                    "end": 2408
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 2413,
                      "end": 2425
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2413,
                    "end": 2425
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 2430,
                      "end": 2438
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2430,
                    "end": 2438
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "yourMembership",
                    "loc": {
                      "start": 2443,
                      "end": 2457
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 2468,
                            "end": 2470
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2468,
                          "end": 2470
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 2479,
                            "end": 2489
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2479,
                          "end": 2489
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 2498,
                            "end": 2508
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2498,
                          "end": 2508
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isAdmin",
                          "loc": {
                            "start": 2517,
                            "end": 2524
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2517,
                          "end": 2524
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "permissions",
                          "loc": {
                            "start": 2533,
                            "end": 2544
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2533,
                          "end": 2544
                        }
                      }
                    ],
                    "loc": {
                      "start": 2458,
                      "end": 2550
                    }
                  },
                  "loc": {
                    "start": 2443,
                    "end": 2550
                  }
                }
              ],
              "loc": {
                "start": 2319,
                "end": 2552
              }
            },
            "loc": {
              "start": 2315,
              "end": 2552
            }
          }
        ],
        "loc": {
          "start": 2098,
          "end": 2554
        }
      },
      "loc": {
        "start": 2055,
        "end": 2554
      }
    },
    "Organization_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Organization_nav",
        "loc": {
          "start": 2564,
          "end": 2580
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Organization",
          "loc": {
            "start": 2584,
            "end": 2596
          }
        },
        "loc": {
          "start": 2584,
          "end": 2596
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 2599,
                "end": 2601
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2599,
              "end": 2601
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bannerImage",
              "loc": {
                "start": 2602,
                "end": 2613
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2602,
              "end": 2613
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 2614,
                "end": 2620
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2614,
              "end": 2620
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "profileImage",
              "loc": {
                "start": 2621,
                "end": 2633
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2621,
              "end": 2633
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 2634,
                "end": 2637
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canAddMembers",
                    "loc": {
                      "start": 2644,
                      "end": 2657
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2644,
                    "end": 2657
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 2662,
                      "end": 2671
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2662,
                    "end": 2671
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 2676,
                      "end": 2687
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2676,
                    "end": 2687
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 2692,
                      "end": 2701
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2692,
                    "end": 2701
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 2706,
                      "end": 2715
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2706,
                    "end": 2715
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 2720,
                      "end": 2727
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2720,
                    "end": 2727
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 2732,
                      "end": 2744
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2732,
                    "end": 2744
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 2749,
                      "end": 2757
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2749,
                    "end": 2757
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "yourMembership",
                    "loc": {
                      "start": 2762,
                      "end": 2776
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 2787,
                            "end": 2789
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2787,
                          "end": 2789
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 2798,
                            "end": 2808
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2798,
                          "end": 2808
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 2817,
                            "end": 2827
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2817,
                          "end": 2827
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isAdmin",
                          "loc": {
                            "start": 2836,
                            "end": 2843
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2836,
                          "end": 2843
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "permissions",
                          "loc": {
                            "start": 2852,
                            "end": 2863
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2852,
                          "end": 2863
                        }
                      }
                    ],
                    "loc": {
                      "start": 2777,
                      "end": 2869
                    }
                  },
                  "loc": {
                    "start": 2762,
                    "end": 2869
                  }
                }
              ],
              "loc": {
                "start": 2638,
                "end": 2871
              }
            },
            "loc": {
              "start": 2634,
              "end": 2871
            }
          }
        ],
        "loc": {
          "start": 2597,
          "end": 2873
        }
      },
      "loc": {
        "start": 2555,
        "end": 2873
      }
    },
    "Project_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Project_list",
        "loc": {
          "start": 2883,
          "end": 2895
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Project",
          "loc": {
            "start": 2899,
            "end": 2906
          }
        },
        "loc": {
          "start": 2899,
          "end": 2906
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versions",
              "loc": {
                "start": 2909,
                "end": 2917
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 2924,
                      "end": 2936
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 2947,
                            "end": 2949
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2947,
                          "end": 2949
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 2958,
                            "end": 2966
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2958,
                          "end": 2966
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 2975,
                            "end": 2986
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2975,
                          "end": 2986
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 2995,
                            "end": 2999
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2995,
                          "end": 2999
                        }
                      }
                    ],
                    "loc": {
                      "start": 2937,
                      "end": 3005
                    }
                  },
                  "loc": {
                    "start": 2924,
                    "end": 3005
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 3010,
                      "end": 3012
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3010,
                    "end": 3012
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 3017,
                      "end": 3027
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3017,
                    "end": 3027
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 3032,
                      "end": 3042
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3032,
                    "end": 3042
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "directoriesCount",
                    "loc": {
                      "start": 3047,
                      "end": 3063
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3047,
                    "end": 3063
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 3068,
                      "end": 3076
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3068,
                    "end": 3076
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 3081,
                      "end": 3090
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3081,
                    "end": 3090
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 3095,
                      "end": 3107
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3095,
                    "end": 3107
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "runProjectsCount",
                    "loc": {
                      "start": 3112,
                      "end": 3128
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3112,
                    "end": 3128
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "simplicity",
                    "loc": {
                      "start": 3133,
                      "end": 3143
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3133,
                    "end": 3143
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 3148,
                      "end": 3160
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3148,
                    "end": 3160
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 3165,
                      "end": 3177
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3165,
                    "end": 3177
                  }
                }
              ],
              "loc": {
                "start": 2918,
                "end": 3179
              }
            },
            "loc": {
              "start": 2909,
              "end": 3179
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 3180,
                "end": 3182
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3180,
              "end": 3182
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 3183,
                "end": 3193
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3183,
              "end": 3193
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 3194,
                "end": 3204
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3194,
              "end": 3204
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 3205,
                "end": 3214
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3205,
              "end": 3214
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 3215,
                "end": 3226
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3215,
              "end": 3226
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 3227,
                "end": 3233
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Label_list",
                    "loc": {
                      "start": 3243,
                      "end": 3253
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 3240,
                    "end": 3253
                  }
                }
              ],
              "loc": {
                "start": 3234,
                "end": 3255
              }
            },
            "loc": {
              "start": 3227,
              "end": 3255
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 3256,
                "end": 3261
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Organization",
                      "loc": {
                        "start": 3275,
                        "end": 3287
                      }
                    },
                    "loc": {
                      "start": 3275,
                      "end": 3287
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Organization_nav",
                          "loc": {
                            "start": 3301,
                            "end": 3317
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 3298,
                          "end": 3317
                        }
                      }
                    ],
                    "loc": {
                      "start": 3288,
                      "end": 3323
                    }
                  },
                  "loc": {
                    "start": 3268,
                    "end": 3323
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "User",
                      "loc": {
                        "start": 3335,
                        "end": 3339
                      }
                    },
                    "loc": {
                      "start": 3335,
                      "end": 3339
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "User_nav",
                          "loc": {
                            "start": 3353,
                            "end": 3361
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 3350,
                          "end": 3361
                        }
                      }
                    ],
                    "loc": {
                      "start": 3340,
                      "end": 3367
                    }
                  },
                  "loc": {
                    "start": 3328,
                    "end": 3367
                  }
                }
              ],
              "loc": {
                "start": 3262,
                "end": 3369
              }
            },
            "loc": {
              "start": 3256,
              "end": 3369
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 3370,
                "end": 3381
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3370,
              "end": 3381
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 3382,
                "end": 3396
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3382,
              "end": 3396
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 3397,
                "end": 3402
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3397,
              "end": 3402
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 3403,
                "end": 3412
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3403,
              "end": 3412
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 3413,
                "end": 3417
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Tag_list",
                    "loc": {
                      "start": 3427,
                      "end": 3435
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 3424,
                    "end": 3435
                  }
                }
              ],
              "loc": {
                "start": 3418,
                "end": 3437
              }
            },
            "loc": {
              "start": 3413,
              "end": 3437
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 3438,
                "end": 3452
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3438,
              "end": 3452
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 3453,
                "end": 3458
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3453,
              "end": 3458
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 3459,
                "end": 3462
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 3469,
                      "end": 3478
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3469,
                    "end": 3478
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 3483,
                      "end": 3494
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3483,
                    "end": 3494
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canTransfer",
                    "loc": {
                      "start": 3499,
                      "end": 3510
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3499,
                    "end": 3510
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 3515,
                      "end": 3524
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3515,
                    "end": 3524
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 3529,
                      "end": 3536
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3529,
                    "end": 3536
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 3541,
                      "end": 3549
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3541,
                    "end": 3549
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 3554,
                      "end": 3566
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3554,
                    "end": 3566
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 3571,
                      "end": 3579
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3571,
                    "end": 3579
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 3584,
                      "end": 3592
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3584,
                    "end": 3592
                  }
                }
              ],
              "loc": {
                "start": 3463,
                "end": 3594
              }
            },
            "loc": {
              "start": 3459,
              "end": 3594
            }
          }
        ],
        "loc": {
          "start": 2907,
          "end": 3596
        }
      },
      "loc": {
        "start": 2874,
        "end": 3596
      }
    },
    "Project_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Project_nav",
        "loc": {
          "start": 3606,
          "end": 3617
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Project",
          "loc": {
            "start": 3621,
            "end": 3628
          }
        },
        "loc": {
          "start": 3621,
          "end": 3628
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 3631,
                "end": 3633
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3631,
              "end": 3633
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 3634,
                "end": 3643
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3634,
              "end": 3643
            }
          }
        ],
        "loc": {
          "start": 3629,
          "end": 3645
        }
      },
      "loc": {
        "start": 3597,
        "end": 3645
      }
    },
    "Question_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Question_list",
        "loc": {
          "start": 3655,
          "end": 3668
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Question",
          "loc": {
            "start": 3672,
            "end": 3680
          }
        },
        "loc": {
          "start": 3672,
          "end": 3680
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 3683,
                "end": 3695
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 3702,
                      "end": 3704
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3702,
                    "end": 3704
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 3709,
                      "end": 3717
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3709,
                    "end": 3717
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 3722,
                      "end": 3733
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3722,
                    "end": 3733
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 3738,
                      "end": 3742
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3738,
                    "end": 3742
                  }
                }
              ],
              "loc": {
                "start": 3696,
                "end": 3744
              }
            },
            "loc": {
              "start": 3683,
              "end": 3744
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 3745,
                "end": 3747
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3745,
              "end": 3747
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 3748,
                "end": 3758
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3748,
              "end": 3758
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 3759,
                "end": 3769
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3759,
              "end": 3769
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "createdBy",
              "loc": {
                "start": 3770,
                "end": 3779
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 3786,
                      "end": 3788
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3786,
                    "end": 3788
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "bannerImage",
                    "loc": {
                      "start": 3793,
                      "end": 3804
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3793,
                    "end": 3804
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "handle",
                    "loc": {
                      "start": 3809,
                      "end": 3815
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3809,
                    "end": 3815
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBot",
                    "loc": {
                      "start": 3820,
                      "end": 3825
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3820,
                    "end": 3825
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 3830,
                      "end": 3834
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3830,
                    "end": 3834
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "profileImage",
                    "loc": {
                      "start": 3839,
                      "end": 3851
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3839,
                    "end": 3851
                  }
                }
              ],
              "loc": {
                "start": 3780,
                "end": 3853
              }
            },
            "loc": {
              "start": 3770,
              "end": 3853
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "hasAcceptedAnswer",
              "loc": {
                "start": 3854,
                "end": 3871
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3854,
              "end": 3871
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 3872,
                "end": 3881
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3872,
              "end": 3881
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 3882,
                "end": 3887
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3882,
              "end": 3887
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 3888,
                "end": 3897
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3888,
              "end": 3897
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "answersCount",
              "loc": {
                "start": 3898,
                "end": 3910
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3898,
              "end": 3910
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "commentsCount",
              "loc": {
                "start": 3911,
                "end": 3924
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3911,
              "end": 3924
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 3925,
                "end": 3937
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3925,
              "end": 3937
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "forObject",
              "loc": {
                "start": 3938,
                "end": 3947
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Api",
                      "loc": {
                        "start": 3961,
                        "end": 3964
                      }
                    },
                    "loc": {
                      "start": 3961,
                      "end": 3964
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Api_nav",
                          "loc": {
                            "start": 3978,
                            "end": 3985
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 3975,
                          "end": 3985
                        }
                      }
                    ],
                    "loc": {
                      "start": 3965,
                      "end": 3991
                    }
                  },
                  "loc": {
                    "start": 3954,
                    "end": 3991
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Note",
                      "loc": {
                        "start": 4003,
                        "end": 4007
                      }
                    },
                    "loc": {
                      "start": 4003,
                      "end": 4007
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Note_nav",
                          "loc": {
                            "start": 4021,
                            "end": 4029
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4018,
                          "end": 4029
                        }
                      }
                    ],
                    "loc": {
                      "start": 4008,
                      "end": 4035
                    }
                  },
                  "loc": {
                    "start": 3996,
                    "end": 4035
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Organization",
                      "loc": {
                        "start": 4047,
                        "end": 4059
                      }
                    },
                    "loc": {
                      "start": 4047,
                      "end": 4059
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Organization_nav",
                          "loc": {
                            "start": 4073,
                            "end": 4089
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4070,
                          "end": 4089
                        }
                      }
                    ],
                    "loc": {
                      "start": 4060,
                      "end": 4095
                    }
                  },
                  "loc": {
                    "start": 4040,
                    "end": 4095
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Project",
                      "loc": {
                        "start": 4107,
                        "end": 4114
                      }
                    },
                    "loc": {
                      "start": 4107,
                      "end": 4114
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Project_nav",
                          "loc": {
                            "start": 4128,
                            "end": 4139
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4125,
                          "end": 4139
                        }
                      }
                    ],
                    "loc": {
                      "start": 4115,
                      "end": 4145
                    }
                  },
                  "loc": {
                    "start": 4100,
                    "end": 4145
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Routine",
                      "loc": {
                        "start": 4157,
                        "end": 4164
                      }
                    },
                    "loc": {
                      "start": 4157,
                      "end": 4164
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Routine_nav",
                          "loc": {
                            "start": 4178,
                            "end": 4189
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4175,
                          "end": 4189
                        }
                      }
                    ],
                    "loc": {
                      "start": 4165,
                      "end": 4195
                    }
                  },
                  "loc": {
                    "start": 4150,
                    "end": 4195
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "SmartContract",
                      "loc": {
                        "start": 4207,
                        "end": 4220
                      }
                    },
                    "loc": {
                      "start": 4207,
                      "end": 4220
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "SmartContract_nav",
                          "loc": {
                            "start": 4234,
                            "end": 4251
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4231,
                          "end": 4251
                        }
                      }
                    ],
                    "loc": {
                      "start": 4221,
                      "end": 4257
                    }
                  },
                  "loc": {
                    "start": 4200,
                    "end": 4257
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Standard",
                      "loc": {
                        "start": 4269,
                        "end": 4277
                      }
                    },
                    "loc": {
                      "start": 4269,
                      "end": 4277
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Standard_nav",
                          "loc": {
                            "start": 4291,
                            "end": 4303
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4288,
                          "end": 4303
                        }
                      }
                    ],
                    "loc": {
                      "start": 4278,
                      "end": 4309
                    }
                  },
                  "loc": {
                    "start": 4262,
                    "end": 4309
                  }
                }
              ],
              "loc": {
                "start": 3948,
                "end": 4311
              }
            },
            "loc": {
              "start": 3938,
              "end": 4311
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 4312,
                "end": 4316
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Tag_list",
                    "loc": {
                      "start": 4326,
                      "end": 4334
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4323,
                    "end": 4334
                  }
                }
              ],
              "loc": {
                "start": 4317,
                "end": 4336
              }
            },
            "loc": {
              "start": 4312,
              "end": 4336
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 4337,
                "end": 4340
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 4347,
                      "end": 4355
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4347,
                    "end": 4355
                  }
                }
              ],
              "loc": {
                "start": 4341,
                "end": 4357
              }
            },
            "loc": {
              "start": 4337,
              "end": 4357
            }
          }
        ],
        "loc": {
          "start": 3681,
          "end": 4359
        }
      },
      "loc": {
        "start": 3646,
        "end": 4359
      }
    },
    "Routine_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Routine_list",
        "loc": {
          "start": 4369,
          "end": 4381
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Routine",
          "loc": {
            "start": 4385,
            "end": 4392
          }
        },
        "loc": {
          "start": 4385,
          "end": 4392
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versions",
              "loc": {
                "start": 4395,
                "end": 4403
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 4410,
                      "end": 4422
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 4433,
                            "end": 4435
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4433,
                          "end": 4435
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 4444,
                            "end": 4452
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4444,
                          "end": 4452
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 4461,
                            "end": 4472
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4461,
                          "end": 4472
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "instructions",
                          "loc": {
                            "start": 4481,
                            "end": 4493
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4481,
                          "end": 4493
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 4502,
                            "end": 4506
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4502,
                          "end": 4506
                        }
                      }
                    ],
                    "loc": {
                      "start": 4423,
                      "end": 4512
                    }
                  },
                  "loc": {
                    "start": 4410,
                    "end": 4512
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 4517,
                      "end": 4519
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4517,
                    "end": 4519
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 4524,
                      "end": 4534
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4524,
                    "end": 4534
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 4539,
                      "end": 4549
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4539,
                    "end": 4549
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "completedAt",
                    "loc": {
                      "start": 4554,
                      "end": 4565
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4554,
                    "end": 4565
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isAutomatable",
                    "loc": {
                      "start": 4570,
                      "end": 4583
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4570,
                    "end": 4583
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isComplete",
                    "loc": {
                      "start": 4588,
                      "end": 4598
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4588,
                    "end": 4598
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isDeleted",
                    "loc": {
                      "start": 4603,
                      "end": 4612
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4603,
                    "end": 4612
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 4617,
                      "end": 4625
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4617,
                    "end": 4625
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 4630,
                      "end": 4639
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4630,
                    "end": 4639
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "simplicity",
                    "loc": {
                      "start": 4644,
                      "end": 4654
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4644,
                    "end": 4654
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "timesStarted",
                    "loc": {
                      "start": 4659,
                      "end": 4671
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4659,
                    "end": 4671
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "timesCompleted",
                    "loc": {
                      "start": 4676,
                      "end": 4690
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4676,
                    "end": 4690
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "smartContractCallData",
                    "loc": {
                      "start": 4695,
                      "end": 4716
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4695,
                    "end": 4716
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "apiCallData",
                    "loc": {
                      "start": 4721,
                      "end": 4732
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4721,
                    "end": 4732
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 4737,
                      "end": 4749
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4737,
                    "end": 4749
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 4754,
                      "end": 4766
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4754,
                    "end": 4766
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "commentsCount",
                    "loc": {
                      "start": 4771,
                      "end": 4784
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4771,
                    "end": 4784
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "directoryListingsCount",
                    "loc": {
                      "start": 4789,
                      "end": 4811
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4789,
                    "end": 4811
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "forksCount",
                    "loc": {
                      "start": 4816,
                      "end": 4826
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4816,
                    "end": 4826
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "inputsCount",
                    "loc": {
                      "start": 4831,
                      "end": 4842
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4831,
                    "end": 4842
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "nodesCount",
                    "loc": {
                      "start": 4847,
                      "end": 4857
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4847,
                    "end": 4857
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "nodeLinksCount",
                    "loc": {
                      "start": 4862,
                      "end": 4876
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4862,
                    "end": 4876
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "outputsCount",
                    "loc": {
                      "start": 4881,
                      "end": 4893
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4881,
                    "end": 4893
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 4898,
                      "end": 4910
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4898,
                    "end": 4910
                  }
                }
              ],
              "loc": {
                "start": 4404,
                "end": 4912
              }
            },
            "loc": {
              "start": 4395,
              "end": 4912
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 4913,
                "end": 4915
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4913,
              "end": 4915
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 4916,
                "end": 4926
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4916,
              "end": 4926
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 4927,
                "end": 4937
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4927,
              "end": 4937
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isInternal",
              "loc": {
                "start": 4938,
                "end": 4948
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4938,
              "end": 4948
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 4949,
                "end": 4958
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4949,
              "end": 4958
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 4959,
                "end": 4970
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4959,
              "end": 4970
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 4971,
                "end": 4977
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Label_list",
                    "loc": {
                      "start": 4987,
                      "end": 4997
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4984,
                    "end": 4997
                  }
                }
              ],
              "loc": {
                "start": 4978,
                "end": 4999
              }
            },
            "loc": {
              "start": 4971,
              "end": 4999
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 5000,
                "end": 5005
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Organization",
                      "loc": {
                        "start": 5019,
                        "end": 5031
                      }
                    },
                    "loc": {
                      "start": 5019,
                      "end": 5031
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Organization_nav",
                          "loc": {
                            "start": 5045,
                            "end": 5061
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 5042,
                          "end": 5061
                        }
                      }
                    ],
                    "loc": {
                      "start": 5032,
                      "end": 5067
                    }
                  },
                  "loc": {
                    "start": 5012,
                    "end": 5067
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "User",
                      "loc": {
                        "start": 5079,
                        "end": 5083
                      }
                    },
                    "loc": {
                      "start": 5079,
                      "end": 5083
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "User_nav",
                          "loc": {
                            "start": 5097,
                            "end": 5105
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 5094,
                          "end": 5105
                        }
                      }
                    ],
                    "loc": {
                      "start": 5084,
                      "end": 5111
                    }
                  },
                  "loc": {
                    "start": 5072,
                    "end": 5111
                  }
                }
              ],
              "loc": {
                "start": 5006,
                "end": 5113
              }
            },
            "loc": {
              "start": 5000,
              "end": 5113
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 5114,
                "end": 5125
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5114,
              "end": 5125
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 5126,
                "end": 5140
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5126,
              "end": 5140
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 5141,
                "end": 5146
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5141,
              "end": 5146
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 5147,
                "end": 5156
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5147,
              "end": 5156
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 5157,
                "end": 5161
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Tag_list",
                    "loc": {
                      "start": 5171,
                      "end": 5179
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 5168,
                    "end": 5179
                  }
                }
              ],
              "loc": {
                "start": 5162,
                "end": 5181
              }
            },
            "loc": {
              "start": 5157,
              "end": 5181
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 5182,
                "end": 5196
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5182,
              "end": 5196
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 5197,
                "end": 5202
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5197,
              "end": 5202
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 5203,
                "end": 5206
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canComment",
                    "loc": {
                      "start": 5213,
                      "end": 5223
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5213,
                    "end": 5223
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 5228,
                      "end": 5237
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5228,
                    "end": 5237
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 5242,
                      "end": 5253
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5242,
                    "end": 5253
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 5258,
                      "end": 5267
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5258,
                    "end": 5267
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 5272,
                      "end": 5279
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5272,
                    "end": 5279
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 5284,
                      "end": 5292
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5284,
                    "end": 5292
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 5297,
                      "end": 5309
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5297,
                    "end": 5309
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 5314,
                      "end": 5322
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5314,
                    "end": 5322
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 5327,
                      "end": 5335
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5327,
                    "end": 5335
                  }
                }
              ],
              "loc": {
                "start": 5207,
                "end": 5337
              }
            },
            "loc": {
              "start": 5203,
              "end": 5337
            }
          }
        ],
        "loc": {
          "start": 4393,
          "end": 5339
        }
      },
      "loc": {
        "start": 4360,
        "end": 5339
      }
    },
    "Routine_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Routine_nav",
        "loc": {
          "start": 5349,
          "end": 5360
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Routine",
          "loc": {
            "start": 5364,
            "end": 5371
          }
        },
        "loc": {
          "start": 5364,
          "end": 5371
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 5374,
                "end": 5376
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5374,
              "end": 5376
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isInternal",
              "loc": {
                "start": 5377,
                "end": 5387
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5377,
              "end": 5387
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 5388,
                "end": 5397
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5388,
              "end": 5397
            }
          }
        ],
        "loc": {
          "start": 5372,
          "end": 5399
        }
      },
      "loc": {
        "start": 5340,
        "end": 5399
      }
    },
    "SmartContract_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "SmartContract_list",
        "loc": {
          "start": 5409,
          "end": 5427
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "SmartContract",
          "loc": {
            "start": 5431,
            "end": 5444
          }
        },
        "loc": {
          "start": 5431,
          "end": 5444
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versions",
              "loc": {
                "start": 5447,
                "end": 5455
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 5462,
                      "end": 5474
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 5485,
                            "end": 5487
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5485,
                          "end": 5487
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 5496,
                            "end": 5504
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5496,
                          "end": 5504
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 5513,
                            "end": 5524
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5513,
                          "end": 5524
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "jsonVariable",
                          "loc": {
                            "start": 5533,
                            "end": 5545
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5533,
                          "end": 5545
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 5554,
                            "end": 5558
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5554,
                          "end": 5558
                        }
                      }
                    ],
                    "loc": {
                      "start": 5475,
                      "end": 5564
                    }
                  },
                  "loc": {
                    "start": 5462,
                    "end": 5564
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 5569,
                      "end": 5571
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5569,
                    "end": 5571
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 5576,
                      "end": 5586
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5576,
                    "end": 5586
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 5591,
                      "end": 5601
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5591,
                    "end": 5601
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isComplete",
                    "loc": {
                      "start": 5606,
                      "end": 5616
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5606,
                    "end": 5616
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isDeleted",
                    "loc": {
                      "start": 5621,
                      "end": 5630
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5621,
                    "end": 5630
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 5635,
                      "end": 5643
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5635,
                    "end": 5643
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 5648,
                      "end": 5657
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5648,
                    "end": 5657
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "default",
                    "loc": {
                      "start": 5662,
                      "end": 5669
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5662,
                    "end": 5669
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "contractType",
                    "loc": {
                      "start": 5674,
                      "end": 5686
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5674,
                    "end": 5686
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "content",
                    "loc": {
                      "start": 5691,
                      "end": 5698
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5691,
                    "end": 5698
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 5703,
                      "end": 5715
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5703,
                    "end": 5715
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 5720,
                      "end": 5732
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5720,
                    "end": 5732
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "commentsCount",
                    "loc": {
                      "start": 5737,
                      "end": 5750
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5737,
                    "end": 5750
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "directoryListingsCount",
                    "loc": {
                      "start": 5755,
                      "end": 5777
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5755,
                    "end": 5777
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "forksCount",
                    "loc": {
                      "start": 5782,
                      "end": 5792
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5782,
                    "end": 5792
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 5797,
                      "end": 5809
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5797,
                    "end": 5809
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 5814,
                      "end": 5817
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canComment",
                          "loc": {
                            "start": 5828,
                            "end": 5838
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5828,
                          "end": 5838
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canCopy",
                          "loc": {
                            "start": 5847,
                            "end": 5854
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5847,
                          "end": 5854
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canDelete",
                          "loc": {
                            "start": 5863,
                            "end": 5872
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5863,
                          "end": 5872
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canReport",
                          "loc": {
                            "start": 5881,
                            "end": 5890
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5881,
                          "end": 5890
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 5899,
                            "end": 5908
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5899,
                          "end": 5908
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUse",
                          "loc": {
                            "start": 5917,
                            "end": 5923
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5917,
                          "end": 5923
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 5932,
                            "end": 5939
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5932,
                          "end": 5939
                        }
                      }
                    ],
                    "loc": {
                      "start": 5818,
                      "end": 5945
                    }
                  },
                  "loc": {
                    "start": 5814,
                    "end": 5945
                  }
                }
              ],
              "loc": {
                "start": 5456,
                "end": 5947
              }
            },
            "loc": {
              "start": 5447,
              "end": 5947
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 5948,
                "end": 5950
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5948,
              "end": 5950
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 5951,
                "end": 5961
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5951,
              "end": 5961
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 5962,
                "end": 5972
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5962,
              "end": 5972
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 5973,
                "end": 5982
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5973,
              "end": 5982
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 5983,
                "end": 5994
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5983,
              "end": 5994
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 5995,
                "end": 6001
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Label_list",
                    "loc": {
                      "start": 6011,
                      "end": 6021
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 6008,
                    "end": 6021
                  }
                }
              ],
              "loc": {
                "start": 6002,
                "end": 6023
              }
            },
            "loc": {
              "start": 5995,
              "end": 6023
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 6024,
                "end": 6029
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Organization",
                      "loc": {
                        "start": 6043,
                        "end": 6055
                      }
                    },
                    "loc": {
                      "start": 6043,
                      "end": 6055
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Organization_nav",
                          "loc": {
                            "start": 6069,
                            "end": 6085
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 6066,
                          "end": 6085
                        }
                      }
                    ],
                    "loc": {
                      "start": 6056,
                      "end": 6091
                    }
                  },
                  "loc": {
                    "start": 6036,
                    "end": 6091
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "User",
                      "loc": {
                        "start": 6103,
                        "end": 6107
                      }
                    },
                    "loc": {
                      "start": 6103,
                      "end": 6107
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "User_nav",
                          "loc": {
                            "start": 6121,
                            "end": 6129
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 6118,
                          "end": 6129
                        }
                      }
                    ],
                    "loc": {
                      "start": 6108,
                      "end": 6135
                    }
                  },
                  "loc": {
                    "start": 6096,
                    "end": 6135
                  }
                }
              ],
              "loc": {
                "start": 6030,
                "end": 6137
              }
            },
            "loc": {
              "start": 6024,
              "end": 6137
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 6138,
                "end": 6149
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6138,
              "end": 6149
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 6150,
                "end": 6164
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6150,
              "end": 6164
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 6165,
                "end": 6170
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6165,
              "end": 6170
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 6171,
                "end": 6180
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6171,
              "end": 6180
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 6181,
                "end": 6185
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Tag_list",
                    "loc": {
                      "start": 6195,
                      "end": 6203
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 6192,
                    "end": 6203
                  }
                }
              ],
              "loc": {
                "start": 6186,
                "end": 6205
              }
            },
            "loc": {
              "start": 6181,
              "end": 6205
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 6206,
                "end": 6220
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6206,
              "end": 6220
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 6221,
                "end": 6226
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6221,
              "end": 6226
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 6227,
                "end": 6230
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 6237,
                      "end": 6246
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6237,
                    "end": 6246
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 6251,
                      "end": 6262
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6251,
                    "end": 6262
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canTransfer",
                    "loc": {
                      "start": 6267,
                      "end": 6278
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6267,
                    "end": 6278
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 6283,
                      "end": 6292
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6283,
                    "end": 6292
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 6297,
                      "end": 6304
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6297,
                    "end": 6304
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 6309,
                      "end": 6317
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6309,
                    "end": 6317
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 6322,
                      "end": 6334
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6322,
                    "end": 6334
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 6339,
                      "end": 6347
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6339,
                    "end": 6347
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 6352,
                      "end": 6360
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6352,
                    "end": 6360
                  }
                }
              ],
              "loc": {
                "start": 6231,
                "end": 6362
              }
            },
            "loc": {
              "start": 6227,
              "end": 6362
            }
          }
        ],
        "loc": {
          "start": 5445,
          "end": 6364
        }
      },
      "loc": {
        "start": 5400,
        "end": 6364
      }
    },
    "SmartContract_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "SmartContract_nav",
        "loc": {
          "start": 6374,
          "end": 6391
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "SmartContract",
          "loc": {
            "start": 6395,
            "end": 6408
          }
        },
        "loc": {
          "start": 6395,
          "end": 6408
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 6411,
                "end": 6413
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6411,
              "end": 6413
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 6414,
                "end": 6423
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6414,
              "end": 6423
            }
          }
        ],
        "loc": {
          "start": 6409,
          "end": 6425
        }
      },
      "loc": {
        "start": 6365,
        "end": 6425
      }
    },
    "Standard_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Standard_list",
        "loc": {
          "start": 6435,
          "end": 6448
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Standard",
          "loc": {
            "start": 6452,
            "end": 6460
          }
        },
        "loc": {
          "start": 6452,
          "end": 6460
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versions",
              "loc": {
                "start": 6463,
                "end": 6471
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 6478,
                      "end": 6490
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 6501,
                            "end": 6503
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6501,
                          "end": 6503
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 6512,
                            "end": 6520
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6512,
                          "end": 6520
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 6529,
                            "end": 6540
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6529,
                          "end": 6540
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "jsonVariable",
                          "loc": {
                            "start": 6549,
                            "end": 6561
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6549,
                          "end": 6561
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 6570,
                            "end": 6574
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6570,
                          "end": 6574
                        }
                      }
                    ],
                    "loc": {
                      "start": 6491,
                      "end": 6580
                    }
                  },
                  "loc": {
                    "start": 6478,
                    "end": 6580
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 6585,
                      "end": 6587
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6585,
                    "end": 6587
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 6592,
                      "end": 6602
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6592,
                    "end": 6602
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 6607,
                      "end": 6617
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6607,
                    "end": 6617
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isComplete",
                    "loc": {
                      "start": 6622,
                      "end": 6632
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6622,
                    "end": 6632
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isFile",
                    "loc": {
                      "start": 6637,
                      "end": 6643
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6637,
                    "end": 6643
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 6648,
                      "end": 6656
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6648,
                    "end": 6656
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 6661,
                      "end": 6670
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6661,
                    "end": 6670
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "default",
                    "loc": {
                      "start": 6675,
                      "end": 6682
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6675,
                    "end": 6682
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "standardType",
                    "loc": {
                      "start": 6687,
                      "end": 6699
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6687,
                    "end": 6699
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "props",
                    "loc": {
                      "start": 6704,
                      "end": 6709
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6704,
                    "end": 6709
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "yup",
                    "loc": {
                      "start": 6714,
                      "end": 6717
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6714,
                    "end": 6717
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 6722,
                      "end": 6734
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6722,
                    "end": 6734
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 6739,
                      "end": 6751
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6739,
                    "end": 6751
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "commentsCount",
                    "loc": {
                      "start": 6756,
                      "end": 6769
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6756,
                    "end": 6769
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "directoryListingsCount",
                    "loc": {
                      "start": 6774,
                      "end": 6796
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6774,
                    "end": 6796
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "forksCount",
                    "loc": {
                      "start": 6801,
                      "end": 6811
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6801,
                    "end": 6811
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 6816,
                      "end": 6828
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6816,
                    "end": 6828
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 6833,
                      "end": 6836
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canComment",
                          "loc": {
                            "start": 6847,
                            "end": 6857
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6847,
                          "end": 6857
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canCopy",
                          "loc": {
                            "start": 6866,
                            "end": 6873
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6866,
                          "end": 6873
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canDelete",
                          "loc": {
                            "start": 6882,
                            "end": 6891
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6882,
                          "end": 6891
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canReport",
                          "loc": {
                            "start": 6900,
                            "end": 6909
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6900,
                          "end": 6909
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 6918,
                            "end": 6927
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6918,
                          "end": 6927
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUse",
                          "loc": {
                            "start": 6936,
                            "end": 6942
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6936,
                          "end": 6942
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 6951,
                            "end": 6958
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6951,
                          "end": 6958
                        }
                      }
                    ],
                    "loc": {
                      "start": 6837,
                      "end": 6964
                    }
                  },
                  "loc": {
                    "start": 6833,
                    "end": 6964
                  }
                }
              ],
              "loc": {
                "start": 6472,
                "end": 6966
              }
            },
            "loc": {
              "start": 6463,
              "end": 6966
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 6967,
                "end": 6969
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6967,
              "end": 6969
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 6970,
                "end": 6980
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6970,
              "end": 6980
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 6981,
                "end": 6991
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6981,
              "end": 6991
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 6992,
                "end": 7001
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6992,
              "end": 7001
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 7002,
                "end": 7013
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7002,
              "end": 7013
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 7014,
                "end": 7020
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Label_list",
                    "loc": {
                      "start": 7030,
                      "end": 7040
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 7027,
                    "end": 7040
                  }
                }
              ],
              "loc": {
                "start": 7021,
                "end": 7042
              }
            },
            "loc": {
              "start": 7014,
              "end": 7042
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 7043,
                "end": 7048
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Organization",
                      "loc": {
                        "start": 7062,
                        "end": 7074
                      }
                    },
                    "loc": {
                      "start": 7062,
                      "end": 7074
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Organization_nav",
                          "loc": {
                            "start": 7088,
                            "end": 7104
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 7085,
                          "end": 7104
                        }
                      }
                    ],
                    "loc": {
                      "start": 7075,
                      "end": 7110
                    }
                  },
                  "loc": {
                    "start": 7055,
                    "end": 7110
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "User",
                      "loc": {
                        "start": 7122,
                        "end": 7126
                      }
                    },
                    "loc": {
                      "start": 7122,
                      "end": 7126
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "User_nav",
                          "loc": {
                            "start": 7140,
                            "end": 7148
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 7137,
                          "end": 7148
                        }
                      }
                    ],
                    "loc": {
                      "start": 7127,
                      "end": 7154
                    }
                  },
                  "loc": {
                    "start": 7115,
                    "end": 7154
                  }
                }
              ],
              "loc": {
                "start": 7049,
                "end": 7156
              }
            },
            "loc": {
              "start": 7043,
              "end": 7156
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 7157,
                "end": 7168
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7157,
              "end": 7168
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 7169,
                "end": 7183
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7169,
              "end": 7183
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 7184,
                "end": 7189
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7184,
              "end": 7189
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 7190,
                "end": 7199
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7190,
              "end": 7199
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 7200,
                "end": 7204
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Tag_list",
                    "loc": {
                      "start": 7214,
                      "end": 7222
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 7211,
                    "end": 7222
                  }
                }
              ],
              "loc": {
                "start": 7205,
                "end": 7224
              }
            },
            "loc": {
              "start": 7200,
              "end": 7224
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 7225,
                "end": 7239
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7225,
              "end": 7239
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 7240,
                "end": 7245
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7240,
              "end": 7245
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 7246,
                "end": 7249
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 7256,
                      "end": 7265
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7256,
                    "end": 7265
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 7270,
                      "end": 7281
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7270,
                    "end": 7281
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canTransfer",
                    "loc": {
                      "start": 7286,
                      "end": 7297
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7286,
                    "end": 7297
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 7302,
                      "end": 7311
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7302,
                    "end": 7311
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 7316,
                      "end": 7323
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7316,
                    "end": 7323
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 7328,
                      "end": 7336
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7328,
                    "end": 7336
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 7341,
                      "end": 7353
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7341,
                    "end": 7353
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 7358,
                      "end": 7366
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7358,
                    "end": 7366
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 7371,
                      "end": 7379
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7371,
                    "end": 7379
                  }
                }
              ],
              "loc": {
                "start": 7250,
                "end": 7381
              }
            },
            "loc": {
              "start": 7246,
              "end": 7381
            }
          }
        ],
        "loc": {
          "start": 6461,
          "end": 7383
        }
      },
      "loc": {
        "start": 6426,
        "end": 7383
      }
    },
    "Standard_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Standard_nav",
        "loc": {
          "start": 7393,
          "end": 7405
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Standard",
          "loc": {
            "start": 7409,
            "end": 7417
          }
        },
        "loc": {
          "start": 7409,
          "end": 7417
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 7420,
                "end": 7422
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7420,
              "end": 7422
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 7423,
                "end": 7432
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7423,
              "end": 7432
            }
          }
        ],
        "loc": {
          "start": 7418,
          "end": 7434
        }
      },
      "loc": {
        "start": 7384,
        "end": 7434
      }
    },
    "Tag_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Tag_list",
        "loc": {
          "start": 7444,
          "end": 7452
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Tag",
          "loc": {
            "start": 7456,
            "end": 7459
          }
        },
        "loc": {
          "start": 7456,
          "end": 7459
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 7462,
                "end": 7464
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7462,
              "end": 7464
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 7465,
                "end": 7475
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7465,
              "end": 7475
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tag",
              "loc": {
                "start": 7476,
                "end": 7479
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7476,
              "end": 7479
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 7480,
                "end": 7489
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7480,
              "end": 7489
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 7490,
                "end": 7502
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 7509,
                      "end": 7511
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7509,
                    "end": 7511
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 7516,
                      "end": 7524
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7516,
                    "end": 7524
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 7529,
                      "end": 7540
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7529,
                    "end": 7540
                  }
                }
              ],
              "loc": {
                "start": 7503,
                "end": 7542
              }
            },
            "loc": {
              "start": 7490,
              "end": 7542
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 7543,
                "end": 7546
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isOwn",
                    "loc": {
                      "start": 7553,
                      "end": 7558
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7553,
                    "end": 7558
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 7563,
                      "end": 7575
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7563,
                    "end": 7575
                  }
                }
              ],
              "loc": {
                "start": 7547,
                "end": 7577
              }
            },
            "loc": {
              "start": 7543,
              "end": 7577
            }
          }
        ],
        "loc": {
          "start": 7460,
          "end": 7579
        }
      },
      "loc": {
        "start": 7435,
        "end": 7579
      }
    },
    "User_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "User_list",
        "loc": {
          "start": 7589,
          "end": 7598
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "User",
          "loc": {
            "start": 7602,
            "end": 7606
          }
        },
        "loc": {
          "start": 7602,
          "end": 7606
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 7609,
                "end": 7621
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 7628,
                      "end": 7630
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7628,
                    "end": 7630
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 7635,
                      "end": 7643
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7635,
                    "end": 7643
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "bio",
                    "loc": {
                      "start": 7648,
                      "end": 7651
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7648,
                    "end": 7651
                  }
                }
              ],
              "loc": {
                "start": 7622,
                "end": 7653
              }
            },
            "loc": {
              "start": 7609,
              "end": 7653
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 7654,
                "end": 7656
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7654,
              "end": 7656
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 7657,
                "end": 7667
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7657,
              "end": 7667
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bannerImage",
              "loc": {
                "start": 7668,
                "end": 7679
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7668,
              "end": 7679
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 7680,
                "end": 7686
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7680,
              "end": 7686
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBot",
              "loc": {
                "start": 7687,
                "end": 7692
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7687,
              "end": 7692
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 7693,
                "end": 7697
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7693,
              "end": 7697
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "profileImage",
              "loc": {
                "start": 7698,
                "end": 7710
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7698,
              "end": 7710
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 7711,
                "end": 7720
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7711,
              "end": 7720
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsReceivedCount",
              "loc": {
                "start": 7721,
                "end": 7741
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7721,
              "end": 7741
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 7742,
                "end": 7745
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 7752,
                      "end": 7761
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7752,
                    "end": 7761
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 7766,
                      "end": 7775
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7766,
                    "end": 7775
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 7780,
                      "end": 7789
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7780,
                    "end": 7789
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 7794,
                      "end": 7806
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7794,
                    "end": 7806
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 7811,
                      "end": 7819
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7811,
                    "end": 7819
                  }
                }
              ],
              "loc": {
                "start": 7746,
                "end": 7821
              }
            },
            "loc": {
              "start": 7742,
              "end": 7821
            }
          }
        ],
        "loc": {
          "start": 7607,
          "end": 7823
        }
      },
      "loc": {
        "start": 7580,
        "end": 7823
      }
    },
    "User_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "User_nav",
        "loc": {
          "start": 7833,
          "end": 7841
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "User",
          "loc": {
            "start": 7845,
            "end": 7849
          }
        },
        "loc": {
          "start": 7845,
          "end": 7849
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 7852,
                "end": 7854
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7852,
              "end": 7854
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bannerImage",
              "loc": {
                "start": 7855,
                "end": 7866
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7855,
              "end": 7866
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 7867,
                "end": 7873
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7867,
              "end": 7873
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBot",
              "loc": {
                "start": 7874,
                "end": 7879
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7874,
              "end": 7879
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 7880,
                "end": 7884
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7880,
              "end": 7884
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "profileImage",
              "loc": {
                "start": 7885,
                "end": 7897
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7885,
              "end": 7897
            }
          }
        ],
        "loc": {
          "start": 7850,
          "end": 7899
        }
      },
      "loc": {
        "start": 7824,
        "end": 7899
      }
    }
  },
  "rootValue": {},
  "operation": {
    "kind": "OperationDefinition",
    "operation": "query",
    "name": {
      "kind": "Name",
      "value": "populars",
      "loc": {
        "start": 7907,
        "end": 7915
      }
    },
    "variableDefinitions": [
      {
        "kind": "VariableDefinition",
        "variable": {
          "kind": "Variable",
          "name": {
            "kind": "Name",
            "value": "input",
            "loc": {
              "start": 7917,
              "end": 7922
            }
          },
          "loc": {
            "start": 7916,
            "end": 7922
          }
        },
        "type": {
          "kind": "NonNullType",
          "type": {
            "kind": "NamedType",
            "name": {
              "kind": "Name",
              "value": "PopularSearchInput",
              "loc": {
                "start": 7924,
                "end": 7942
              }
            },
            "loc": {
              "start": 7924,
              "end": 7942
            }
          },
          "loc": {
            "start": 7924,
            "end": 7943
          }
        },
        "directives": [],
        "loc": {
          "start": 7916,
          "end": 7943
        }
      }
    ],
    "directives": [],
    "selectionSet": {
      "kind": "SelectionSet",
      "selections": [
        {
          "kind": "Field",
          "name": {
            "kind": "Name",
            "value": "populars",
            "loc": {
              "start": 7949,
              "end": 7957
            }
          },
          "arguments": [
            {
              "kind": "Argument",
              "name": {
                "kind": "Name",
                "value": "input",
                "loc": {
                  "start": 7958,
                  "end": 7963
                }
              },
              "value": {
                "kind": "Variable",
                "name": {
                  "kind": "Name",
                  "value": "input",
                  "loc": {
                    "start": 7966,
                    "end": 7971
                  }
                },
                "loc": {
                  "start": 7965,
                  "end": 7971
                }
              },
              "loc": {
                "start": 7958,
                "end": 7971
              }
            }
          ],
          "directives": [],
          "selectionSet": {
            "kind": "SelectionSet",
            "selections": [
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "edges",
                  "loc": {
                    "start": 7979,
                    "end": 7984
                  }
                },
                "arguments": [],
                "directives": [],
                "selectionSet": {
                  "kind": "SelectionSet",
                  "selections": [
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "cursor",
                        "loc": {
                          "start": 7995,
                          "end": 8001
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 7995,
                        "end": 8001
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "node",
                        "loc": {
                          "start": 8010,
                          "end": 8014
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "selectionSet": {
                        "kind": "SelectionSet",
                        "selections": [
                          {
                            "kind": "InlineFragment",
                            "typeCondition": {
                              "kind": "NamedType",
                              "name": {
                                "kind": "Name",
                                "value": "Api",
                                "loc": {
                                  "start": 8036,
                                  "end": 8039
                                }
                              },
                              "loc": {
                                "start": 8036,
                                "end": 8039
                              }
                            },
                            "directives": [],
                            "selectionSet": {
                              "kind": "SelectionSet",
                              "selections": [
                                {
                                  "kind": "FragmentSpread",
                                  "name": {
                                    "kind": "Name",
                                    "value": "Api_list",
                                    "loc": {
                                      "start": 8061,
                                      "end": 8069
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 8058,
                                    "end": 8069
                                  }
                                }
                              ],
                              "loc": {
                                "start": 8040,
                                "end": 8083
                              }
                            },
                            "loc": {
                              "start": 8029,
                              "end": 8083
                            }
                          },
                          {
                            "kind": "InlineFragment",
                            "typeCondition": {
                              "kind": "NamedType",
                              "name": {
                                "kind": "Name",
                                "value": "Note",
                                "loc": {
                                  "start": 8103,
                                  "end": 8107
                                }
                              },
                              "loc": {
                                "start": 8103,
                                "end": 8107
                              }
                            },
                            "directives": [],
                            "selectionSet": {
                              "kind": "SelectionSet",
                              "selections": [
                                {
                                  "kind": "FragmentSpread",
                                  "name": {
                                    "kind": "Name",
                                    "value": "Note_list",
                                    "loc": {
                                      "start": 8129,
                                      "end": 8138
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 8126,
                                    "end": 8138
                                  }
                                }
                              ],
                              "loc": {
                                "start": 8108,
                                "end": 8152
                              }
                            },
                            "loc": {
                              "start": 8096,
                              "end": 8152
                            }
                          },
                          {
                            "kind": "InlineFragment",
                            "typeCondition": {
                              "kind": "NamedType",
                              "name": {
                                "kind": "Name",
                                "value": "Organization",
                                "loc": {
                                  "start": 8172,
                                  "end": 8184
                                }
                              },
                              "loc": {
                                "start": 8172,
                                "end": 8184
                              }
                            },
                            "directives": [],
                            "selectionSet": {
                              "kind": "SelectionSet",
                              "selections": [
                                {
                                  "kind": "FragmentSpread",
                                  "name": {
                                    "kind": "Name",
                                    "value": "Organization_list",
                                    "loc": {
                                      "start": 8206,
                                      "end": 8223
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 8203,
                                    "end": 8223
                                  }
                                }
                              ],
                              "loc": {
                                "start": 8185,
                                "end": 8237
                              }
                            },
                            "loc": {
                              "start": 8165,
                              "end": 8237
                            }
                          },
                          {
                            "kind": "InlineFragment",
                            "typeCondition": {
                              "kind": "NamedType",
                              "name": {
                                "kind": "Name",
                                "value": "Project",
                                "loc": {
                                  "start": 8257,
                                  "end": 8264
                                }
                              },
                              "loc": {
                                "start": 8257,
                                "end": 8264
                              }
                            },
                            "directives": [],
                            "selectionSet": {
                              "kind": "SelectionSet",
                              "selections": [
                                {
                                  "kind": "FragmentSpread",
                                  "name": {
                                    "kind": "Name",
                                    "value": "Project_list",
                                    "loc": {
                                      "start": 8286,
                                      "end": 8298
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 8283,
                                    "end": 8298
                                  }
                                }
                              ],
                              "loc": {
                                "start": 8265,
                                "end": 8312
                              }
                            },
                            "loc": {
                              "start": 8250,
                              "end": 8312
                            }
                          },
                          {
                            "kind": "InlineFragment",
                            "typeCondition": {
                              "kind": "NamedType",
                              "name": {
                                "kind": "Name",
                                "value": "Question",
                                "loc": {
                                  "start": 8332,
                                  "end": 8340
                                }
                              },
                              "loc": {
                                "start": 8332,
                                "end": 8340
                              }
                            },
                            "directives": [],
                            "selectionSet": {
                              "kind": "SelectionSet",
                              "selections": [
                                {
                                  "kind": "FragmentSpread",
                                  "name": {
                                    "kind": "Name",
                                    "value": "Question_list",
                                    "loc": {
                                      "start": 8362,
                                      "end": 8375
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 8359,
                                    "end": 8375
                                  }
                                }
                              ],
                              "loc": {
                                "start": 8341,
                                "end": 8389
                              }
                            },
                            "loc": {
                              "start": 8325,
                              "end": 8389
                            }
                          },
                          {
                            "kind": "InlineFragment",
                            "typeCondition": {
                              "kind": "NamedType",
                              "name": {
                                "kind": "Name",
                                "value": "Routine",
                                "loc": {
                                  "start": 8409,
                                  "end": 8416
                                }
                              },
                              "loc": {
                                "start": 8409,
                                "end": 8416
                              }
                            },
                            "directives": [],
                            "selectionSet": {
                              "kind": "SelectionSet",
                              "selections": [
                                {
                                  "kind": "FragmentSpread",
                                  "name": {
                                    "kind": "Name",
                                    "value": "Routine_list",
                                    "loc": {
                                      "start": 8438,
                                      "end": 8450
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 8435,
                                    "end": 8450
                                  }
                                }
                              ],
                              "loc": {
                                "start": 8417,
                                "end": 8464
                              }
                            },
                            "loc": {
                              "start": 8402,
                              "end": 8464
                            }
                          },
                          {
                            "kind": "InlineFragment",
                            "typeCondition": {
                              "kind": "NamedType",
                              "name": {
                                "kind": "Name",
                                "value": "SmartContract",
                                "loc": {
                                  "start": 8484,
                                  "end": 8497
                                }
                              },
                              "loc": {
                                "start": 8484,
                                "end": 8497
                              }
                            },
                            "directives": [],
                            "selectionSet": {
                              "kind": "SelectionSet",
                              "selections": [
                                {
                                  "kind": "FragmentSpread",
                                  "name": {
                                    "kind": "Name",
                                    "value": "SmartContract_list",
                                    "loc": {
                                      "start": 8519,
                                      "end": 8537
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 8516,
                                    "end": 8537
                                  }
                                }
                              ],
                              "loc": {
                                "start": 8498,
                                "end": 8551
                              }
                            },
                            "loc": {
                              "start": 8477,
                              "end": 8551
                            }
                          },
                          {
                            "kind": "InlineFragment",
                            "typeCondition": {
                              "kind": "NamedType",
                              "name": {
                                "kind": "Name",
                                "value": "Standard",
                                "loc": {
                                  "start": 8571,
                                  "end": 8579
                                }
                              },
                              "loc": {
                                "start": 8571,
                                "end": 8579
                              }
                            },
                            "directives": [],
                            "selectionSet": {
                              "kind": "SelectionSet",
                              "selections": [
                                {
                                  "kind": "FragmentSpread",
                                  "name": {
                                    "kind": "Name",
                                    "value": "Standard_list",
                                    "loc": {
                                      "start": 8601,
                                      "end": 8614
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 8598,
                                    "end": 8614
                                  }
                                }
                              ],
                              "loc": {
                                "start": 8580,
                                "end": 8628
                              }
                            },
                            "loc": {
                              "start": 8564,
                              "end": 8628
                            }
                          },
                          {
                            "kind": "InlineFragment",
                            "typeCondition": {
                              "kind": "NamedType",
                              "name": {
                                "kind": "Name",
                                "value": "User",
                                "loc": {
                                  "start": 8648,
                                  "end": 8652
                                }
                              },
                              "loc": {
                                "start": 8648,
                                "end": 8652
                              }
                            },
                            "directives": [],
                            "selectionSet": {
                              "kind": "SelectionSet",
                              "selections": [
                                {
                                  "kind": "FragmentSpread",
                                  "name": {
                                    "kind": "Name",
                                    "value": "User_list",
                                    "loc": {
                                      "start": 8674,
                                      "end": 8683
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 8671,
                                    "end": 8683
                                  }
                                }
                              ],
                              "loc": {
                                "start": 8653,
                                "end": 8697
                              }
                            },
                            "loc": {
                              "start": 8641,
                              "end": 8697
                            }
                          }
                        ],
                        "loc": {
                          "start": 8015,
                          "end": 8707
                        }
                      },
                      "loc": {
                        "start": 8010,
                        "end": 8707
                      }
                    }
                  ],
                  "loc": {
                    "start": 7985,
                    "end": 8713
                  }
                },
                "loc": {
                  "start": 7979,
                  "end": 8713
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "pageInfo",
                  "loc": {
                    "start": 8718,
                    "end": 8726
                  }
                },
                "arguments": [],
                "directives": [],
                "selectionSet": {
                  "kind": "SelectionSet",
                  "selections": [
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "hasNextPage",
                        "loc": {
                          "start": 8737,
                          "end": 8748
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 8737,
                        "end": 8748
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorApi",
                        "loc": {
                          "start": 8757,
                          "end": 8769
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 8757,
                        "end": 8769
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorNote",
                        "loc": {
                          "start": 8778,
                          "end": 8791
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 8778,
                        "end": 8791
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorOrganization",
                        "loc": {
                          "start": 8800,
                          "end": 8821
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 8800,
                        "end": 8821
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorProject",
                        "loc": {
                          "start": 8830,
                          "end": 8846
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 8830,
                        "end": 8846
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorQuestion",
                        "loc": {
                          "start": 8855,
                          "end": 8872
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 8855,
                        "end": 8872
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorRoutine",
                        "loc": {
                          "start": 8881,
                          "end": 8897
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 8881,
                        "end": 8897
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorSmartContract",
                        "loc": {
                          "start": 8906,
                          "end": 8928
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 8906,
                        "end": 8928
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorStandard",
                        "loc": {
                          "start": 8937,
                          "end": 8954
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 8937,
                        "end": 8954
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorUser",
                        "loc": {
                          "start": 8963,
                          "end": 8976
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 8963,
                        "end": 8976
                      }
                    }
                  ],
                  "loc": {
                    "start": 8727,
                    "end": 8982
                  }
                },
                "loc": {
                  "start": 8718,
                  "end": 8982
                }
              }
            ],
            "loc": {
              "start": 7973,
              "end": 8986
            }
          },
          "loc": {
            "start": 7949,
            "end": 8986
          }
        }
      ],
      "loc": {
        "start": 7945,
        "end": 8988
      }
    },
    "loc": {
      "start": 7901,
      "end": 8988
    }
  },
  "variableValues": {},
  "path": {
    "key": "popular_findMany"
  }
} as const;
